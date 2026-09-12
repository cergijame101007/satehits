import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import SupplierManager from '@/components/react/SupplierManager';
import {
  createSupplier,
  deleteSupplier,
  listAdminSuppliers,
  reorderSuppliers,
  SupplierApiError,
  updateSupplier,
  uploadSupplierImage,
} from '@/lib/suppliers';
import type { Supplier } from '@/types/supplier';

vi.mock('@/lib/suppliers', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/suppliers')>();
  return {
    ...actual,
    listAdminSuppliers: vi.fn(),
    createSupplier: vi.fn(),
    updateSupplier: vi.fn(),
    deleteSupplier: vi.fn(),
    reorderSuppliers: vi.fn(),
    uploadSupplierImage: vi.fn(),
  };
});

const listMock = vi.mocked(listAdminSuppliers);
const createMock = vi.mocked(createSupplier);
const updateMock = vi.mocked(updateSupplier);
const deleteMock = vi.mocked(deleteSupplier);
const reorderMock = vi.mocked(reorderSuppliers);
const uploadMock = vi.mocked(uploadSupplierImage);

function supplier(overrides: Partial<Supplier>): Supplier {
  return {
    id: 1,
    name: '山下農園',
    description: '無農薬野菜',
    instagram_url: null,
    image_url: null,
    display_order: 1,
    is_active: true,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    ...overrides,
  };
}

const suppliers: Supplier[] = [
  supplier({ id: 2, name: '海辺の魚屋', description: '朝獲れの魚', display_order: 2, is_active: false }),
  supplier({
    id: 1,
    name: '山下農園',
    description: '無農薬野菜',
    display_order: 1,
    instagram_url: 'https://www.instagram.com/yamashita/',
    image_url: 'https://cdn.example.com/1.jpg',
  }),
];

function getSupplierCard(name: string): HTMLElement {
  const card = screen.getByRole('heading', { name }).closest('[draggable]');
  if (!card) throw new Error(`${name} のカードが見つかりません`);
  return card as HTMLElement;
}

function getDialog(): HTMLElement {
  return screen.getByRole('dialog');
}

async function renderLoaded() {
  render(<SupplierManager />);
  await waitFor(() => {
    expect(screen.queryByText('読み込み中...')).not.toBeInTheDocument();
  });
}

describe('SupplierManager', () => {
  beforeEach(() => {
    listMock.mockResolvedValue(suppliers);
    vi.stubGlobal('URL', { ...URL, createObjectURL: vi.fn(() => 'blob:preview') });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it('取得中は読み込み中を表示し、取得後に並び順で一覧を表示する', async () => {
    render(<SupplierManager />);

    expect(screen.getByText('読み込み中...')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '+ 取引先を追加' })).toBeDisabled();

    await waitFor(() => {
      expect(screen.queryByText('読み込み中...')).not.toBeInTheDocument();
    });

    const headings = screen.getAllByRole('heading', { level: 3 }).map((h) => h.textContent);
    expect(headings).toEqual(['山下農園', '海辺の魚屋']);
    expect(within(getSupplierCard('山下農園')).getByText('表示中')).toBeInTheDocument();
    expect(within(getSupplierCard('山下農園')).getByRole('link', { name: /Instagram/ })).toHaveAttribute(
      'href',
      'https://www.instagram.com/yamashita/',
    );
    expect(within(getSupplierCard('海辺の魚屋')).getByText('非表示')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '+ 取引先を追加' })).toBeEnabled();
  });

  it('取引先がなければ空表示にする', async () => {
    listMock.mockResolvedValue([]);
    await renderLoaded();

    expect(screen.getByText('取引先が登録されていません')).toBeInTheDocument();
  });

  it('取得に失敗したらエラーを表示する', async () => {
    listMock.mockRejectedValue(new SupplierApiError({ code: 'INTERNAL_ERROR', message: '取得に失敗しました' }));
    await renderLoaded();

    expect(screen.getByRole('alert')).toHaveTextContent('取得に失敗しました');
  });

  it('追加モーダルで必須項目が空なら保存せずエラーを表示する', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(screen.getByRole('button', { name: '+ 取引先を追加' }));
    const dialog = screen.getByRole('dialog', { name: '取引先を追加' });
    await user.click(within(dialog).getByRole('button', { name: '保存する' }));

    expect(within(dialog).getByText('取引先名を入力してください')).toBeInTheDocument();
    expect(within(dialog).getByText('説明文を入力してください')).toBeInTheDocument();
    expect(createMock).not.toHaveBeenCalled();
  });

  it('Instagram 以外の URL はエラーにする', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(screen.getByRole('button', { name: '+ 取引先を追加' }));
    const dialog = getDialog();
    await user.type(within(dialog).getByPlaceholderText('例: 〇〇農園'), '新しい農園');
    await user.type(within(dialog).getByPlaceholderText('取引先の紹介文を入力'), '説明');
    await user.type(within(dialog).getByPlaceholderText('https://instagram.com/...'), 'https://example.com/x');
    await user.click(within(dialog).getByRole('button', { name: '保存する' }));

    expect(within(dialog).getByText('有効なInstagramのURLを入力してください')).toBeInTheDocument();
    expect(createMock).not.toHaveBeenCalled();
  });

  it('新規作成すると API を呼び、モーダルを閉じて一覧に追加する', async () => {
    const user = userEvent.setup();
    createMock.mockResolvedValue(supplier({ id: 3, name: '新しい農園', description: '説明', display_order: 3 }));
    await renderLoaded();

    await user.click(screen.getByRole('button', { name: '+ 取引先を追加' }));
    const dialog = getDialog();
    await user.type(within(dialog).getByPlaceholderText('例: 〇〇農園'), ' 新しい農園 ');
    await user.type(within(dialog).getByPlaceholderText('取引先の紹介文を入力'), '説明');
    await user.click(within(dialog).getByRole('checkbox'));
    await user.click(within(dialog).getByRole('button', { name: '保存する' }));

    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
    expect(createMock).toHaveBeenCalledWith({
      name: '新しい農園',
      description: '説明',
      instagram_url: null,
      is_active: false,
    });
    expect(uploadMock).not.toHaveBeenCalled();
    expect(screen.getByRole('heading', { name: '新しい農園' })).toBeInTheDocument();
  });

  it('画像を選ぶと作成後にアップロードし、返された画像 URL を一覧に反映する', async () => {
    const user = userEvent.setup();
    createMock.mockResolvedValue(supplier({ id: 3, name: '新しい農園', description: '説明', display_order: 3 }));
    uploadMock.mockResolvedValue('https://cdn.example.com/3.jpg');
    await renderLoaded();

    await user.click(screen.getByRole('button', { name: '+ 取引先を追加' }));
    const dialog = getDialog();
    await user.type(within(dialog).getByPlaceholderText('例: 〇〇農園'), '新しい農園');
    await user.type(within(dialog).getByPlaceholderText('取引先の紹介文を入力'), '説明');

    const file = new File(['x'], 'photo.png', { type: 'image/png' });
    await user.upload(dialog.querySelector('input[type="file"]')!, file);
    expect(within(dialog).getByRole('img', { name: 'プレビュー' })).toHaveAttribute('src', 'blob:preview');

    await user.click(within(dialog).getByRole('button', { name: '保存する' }));

    await waitFor(() => {
      expect(uploadMock).toHaveBeenCalledWith(3, file);
    });
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
    expect(within(getSupplierCard('新しい農園')).getByRole('presentation')).toHaveAttribute(
      'src',
      'https://cdn.example.com/3.jpg',
    );
  });

  it('対応していない画像形式はエラーにしてプレビューしない', async () => {
    // accept 属性で弾かれずにコンポーネント側の検証へ到達させる
    const user = userEvent.setup({ applyAccept: false });
    await renderLoaded();

    await user.click(screen.getByRole('button', { name: '+ 取引先を追加' }));
    const dialog = getDialog();
    const file = new File(['x'], 'photo.gif', { type: 'image/gif' });
    await user.upload(dialog.querySelector('input[type="file"]')!, file);

    expect(within(dialog).getByText('JPEG / PNG / WebP 形式の画像を選択してください')).toBeInTheDocument();
    expect(within(dialog).queryByRole('img')).not.toBeInTheDocument();
  });

  it('編集モーダルは既存の値で開き、保存で更新 API を呼んで一覧に反映する', async () => {
    const user = userEvent.setup();
    updateMock.mockResolvedValue(
      supplier({ id: 1, name: '山下農園（改）', description: '無農薬野菜', display_order: 1 }),
    );
    await renderLoaded();

    await user.click(within(getSupplierCard('山下農園')).getByRole('button', { name: '編集' }));
    const dialog = screen.getByRole('dialog', { name: '取引先を編集' });
    expect(within(dialog).getByDisplayValue('山下農園')).toBeInTheDocument();
    expect(within(dialog).getByDisplayValue('https://www.instagram.com/yamashita/')).toBeInTheDocument();
    expect(within(dialog).getByRole('img', { name: 'プレビュー' })).toHaveAttribute('src', 'https://cdn.example.com/1.jpg');
    expect(within(dialog).getByRole('checkbox')).toBeChecked();

    const nameInput = within(dialog).getByDisplayValue('山下農園');
    await user.clear(nameInput);
    await user.type(nameInput, '山下農園（改）');
    await user.click(within(dialog).getByRole('button', { name: '保存する' }));

    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
    expect(updateMock).toHaveBeenCalledWith(1, {
      name: '山下農園（改）',
      description: '無農薬野菜',
      instagram_url: 'https://www.instagram.com/yamashita/',
      is_active: true,
    });
    expect(createMock).not.toHaveBeenCalled();
    expect(screen.getByRole('heading', { name: '山下農園（改）' })).toBeInTheDocument();
  });

  it('保存の API エラーは details があればフィールドに、なければ全体に表示する', async () => {
    const user = userEvent.setup();
    createMock.mockRejectedValueOnce(
      new SupplierApiError({
        code: 'VALIDATION_ERROR',
        message: '入力内容に誤りがあります',
        details: [{ field: 'name', message: '取引先名が長すぎます' }],
      }),
    );
    await renderLoaded();

    await user.click(screen.getByRole('button', { name: '+ 取引先を追加' }));
    const dialog = getDialog();
    await user.type(within(dialog).getByPlaceholderText('例: 〇〇農園'), '名前');
    await user.type(within(dialog).getByPlaceholderText('取引先の紹介文を入力'), '説明');
    await user.click(within(dialog).getByRole('button', { name: '保存する' }));

    expect(await within(dialog).findByText('取引先名が長すぎます')).toBeInTheDocument();

    createMock.mockRejectedValueOnce(new SupplierApiError({ code: 'CONFLICT', message: '同名の取引先があります' }));
    await user.click(within(dialog).getByRole('button', { name: '保存する' }));

    expect(await within(dialog).findByRole('alert')).toHaveTextContent('同名の取引先があります');
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  it('モーダルをキャンセルすると保存せずに閉じる', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(screen.getByRole('button', { name: '+ 取引先を追加' }));
    await user.click(within(getDialog()).getByRole('button', { name: 'キャンセル' }));

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(createMock).not.toHaveBeenCalled();
  });

  it('削除は確認モーダルを経て API を呼び、一覧から取り除く', async () => {
    const user = userEvent.setup();
    deleteMock.mockResolvedValue(undefined);
    await renderLoaded();

    await user.click(within(getSupplierCard('海辺の魚屋')).getByRole('button', { name: '削除' }));
    const dialog = screen.getByRole('dialog', { name: '取引先を削除' });
    await user.click(within(dialog).getByRole('button', { name: '削除する' }));

    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
    expect(deleteMock).toHaveBeenCalledWith(2);
    expect(screen.queryByRole('heading', { name: '海辺の魚屋' })).not.toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '山下農園' })).toBeInTheDocument();
  });

  it('削除の確認をキャンセルすると API を呼ばない', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(within(getSupplierCard('海辺の魚屋')).getByRole('button', { name: '削除' }));
    await user.click(within(getDialog()).getByRole('button', { name: 'キャンセル' }));

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(deleteMock).not.toHaveBeenCalled();
  });

  it('削除に失敗したらエラーを表示し、一覧は変えない', async () => {
    const user = userEvent.setup();
    deleteMock.mockRejectedValue(new SupplierApiError({ code: 'INTERNAL_ERROR', message: '削除できませんでした' }));
    await renderLoaded();

    await user.click(within(getSupplierCard('海辺の魚屋')).getByRole('button', { name: '削除' }));
    await user.click(within(getDialog()).getByRole('button', { name: '削除する' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('削除できませんでした');
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '海辺の魚屋' })).toBeInTheDocument();
  });

  it('ドラッグ＆ドロップで並び替えると新しい順序で並び順更新 API を呼ぶ', async () => {
    reorderMock.mockImplementation(async (ids) =>
      ids.map((id, i) => ({ ...suppliers.find((s) => s.id === id)!, display_order: i + 1 })),
    );
    await renderLoaded();

    fireEvent.dragStart(getSupplierCard('海辺の魚屋'));
    fireEvent.dragOver(getSupplierCard('山下農園'));
    fireEvent.dragEnd(getSupplierCard('海辺の魚屋'));

    await waitFor(() => {
      expect(reorderMock).toHaveBeenCalledWith([2, 1]);
    });
    const headings = screen.getAllByRole('heading', { level: 3 }).map((h) => h.textContent);
    expect(headings).toEqual(['海辺の魚屋', '山下農園']);
  });

  it('並び順の更新に失敗したらエラーを表示し、一覧を取り直す', async () => {
    reorderMock.mockRejectedValue(new SupplierApiError({ code: 'INTERNAL_ERROR', message: '並び替えに失敗しました' }));
    await renderLoaded();

    fireEvent.dragStart(getSupplierCard('海辺の魚屋'));
    fireEvent.dragOver(getSupplierCard('山下農園'));
    fireEvent.dragEnd(getSupplierCard('海辺の魚屋'));

    expect(await screen.findByRole('alert')).toHaveTextContent('並び替えに失敗しました');
    await waitFor(() => {
      expect(listMock).toHaveBeenCalledTimes(2);
    });
    const headings = screen.getAllByRole('heading', { level: 3 }).map((h) => h.textContent);
    expect(headings).toEqual(['山下農園', '海辺の魚屋']);
  });
});
