import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import ButtonLink from '@/components/react/ui/ButtonLink';

describe('ButtonLink', () => {
  it('href 付きのリンクとして描画される', () => {
    render(<ButtonLink href="/admin/reservations">予約一覧へ</ButtonLink>);

    const link = screen.getByRole('link', { name: '予約一覧へ' });
    expect(link).toHaveAttribute('href', '/admin/reservations');
  });

  it('Button と同じ variant / size のスタイルが適用される', () => {
    render(
      <ButtonLink href="/" variant="ghost" size="sm">
        戻る
      </ButtonLink>,
    );

    const link = screen.getByRole('link', { name: '戻る' });
    expect(link).toHaveClass('border-gray-300', 'text-sm');
  });

  it('target / rel などの a 属性がそのまま渡される', () => {
    render(
      <ButtonLink href="https://example.com" target="_blank" rel="noopener noreferrer">
        外部
      </ButtonLink>,
    );

    const link = screen.getByRole('link', { name: '外部' });
    expect(link).toHaveAttribute('target', '_blank');
    expect(link).toHaveAttribute('rel', 'noopener noreferrer');
  });
});
