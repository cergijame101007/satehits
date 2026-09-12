import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import SupplierList from '@/components/react/SupplierList';
import { listPublicSuppliers, PublicSupplierApiError, type PublicSupplier } from '@/lib/publicSuppliers';
import { createDeferred } from '@/test/deferred';

vi.mock('@/lib/publicSuppliers', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/publicSuppliers')>();
  return { ...actual, listPublicSuppliers: vi.fn() };
});

const listMock = vi.mocked(listPublicSuppliers);

const suppliers: PublicSupplier[] = [
  {
    id: 1,
    name: '山下農園',
    description: '無農薬野菜を育てています',
    instagram_url: 'https://www.instagram.com/yamashita/',
    image_url: 'https://cdn.example.com/1.jpg',
  },
  {
    id: 2,
    name: '海辺の魚屋',
    description: '朝獲れの魚',
    instagram_url: null,
    image_url: null,
  },
];

function getCard(name: string): HTMLElement {
  const card = screen.getByRole('heading', { name }).closest('.rounded-xl');
  if (!card) throw new Error(`${name} のカードが見つかりません`);
  return card as HTMLElement;
}

describe('SupplierList', () => {
  beforeEach(() => {
    listMock.mockResolvedValue(suppliers);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('取得中はスケルトンを表示し、取得後に一覧へ切り替わる', async () => {
    const deferred = createDeferred<PublicSupplier[]>();
    listMock.mockReturnValue(deferred.promise);
    const { container } = render(<SupplierList />);

    expect(container.querySelectorAll('.animate-pulse')).toHaveLength(2);
    expect(screen.queryByRole('heading')).not.toBeInTheDocument();

    deferred.resolve(suppliers);

    expect(await screen.findByRole('heading', { name: '山下農園' })).toBeInTheDocument();
    expect(container.querySelectorAll('.animate-pulse')).toHaveLength(0);
    expect(listMock).toHaveBeenCalledTimes(1);
  });

  it('各取引先の名前・説明・画像・Instagram リンクを表示する', async () => {
    render(<SupplierList />);
    await screen.findByRole('heading', { name: '山下農園' });

    const yamashita = getCard('山下農園');
    expect(within(yamashita).getByText('無農薬野菜を育てています')).toBeInTheDocument();
    expect(within(yamashita).getByRole('img', { name: '山下農園' })).toHaveAttribute(
      'src',
      'https://cdn.example.com/1.jpg',
    );
    const link = within(yamashita).getByRole('link', { name: 'Instagramを見る' });
    expect(link).toHaveAttribute('href', 'https://www.instagram.com/yamashita/');
    expect(link).toHaveAttribute('target', '_blank');
    expect(link).toHaveAttribute('rel', 'noopener noreferrer');
  });

  it('画像や Instagram がない取引先はプレースホルダーを表示し、リンクを出さない', async () => {
    render(<SupplierList />);
    await screen.findByRole('heading', { name: '海辺の魚屋' });

    const fishShop = getCard('海辺の魚屋');
    expect(within(fishShop).queryByRole('img')).not.toBeInTheDocument();
    expect(fishShop.querySelector('svg')).not.toBeNull();
    expect(within(fishShop).queryByRole('link')).not.toBeInTheDocument();
  });

  it('取引先が 0 件なら案内文を表示する', async () => {
    listMock.mockResolvedValue([]);
    render(<SupplierList />);

    expect(await screen.findByText('現在、公開中の取引先はありません')).toBeInTheDocument();
  });

  it('API エラーはそのメッセージを、想定外の例外は汎用メッセージを表示する', async () => {
    listMock.mockRejectedValueOnce(
      new PublicSupplierApiError({ code: 'INTERNAL_ERROR', message: 'サーバー内部でエラーが発生しました' }),
    );
    const { unmount } = render(<SupplierList />);
    expect(await screen.findByText('サーバー内部でエラーが発生しました')).toBeInTheDocument();
    unmount();

    listMock.mockRejectedValueOnce(new TypeError('Failed to fetch'));
    render(<SupplierList />);
    expect(await screen.findByText('取引先一覧の取得に失敗しました')).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.queryByRole('heading')).not.toBeInTheDocument();
    });
  });
});
