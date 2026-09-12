import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import Alert from '@/components/react/ui/Alert';

describe('Alert', () => {
  it('role="alert" で子要素を表示する', () => {
    render(<Alert>エラーが発生しました</Alert>);

    expect(screen.getByRole('alert')).toHaveTextContent('エラーが発生しました');
  });

  it('variant 未指定なら error のスタイルになる', () => {
    render(<Alert>msg</Alert>);

    expect(screen.getByRole('alert')).toHaveClass('text-red-700');
  });

  it('variant に応じたスタイルが適用される', () => {
    const { rerender } = render(<Alert variant="warning">msg</Alert>);
    expect(screen.getByRole('alert')).toHaveClass('text-amber-800');

    rerender(<Alert variant="info">msg</Alert>);
    expect(screen.getByRole('alert')).toHaveClass('text-gray-700');
  });

  it('compact 指定で小さいパディングと文字サイズになる', () => {
    render(<Alert compact>msg</Alert>);

    const alert = screen.getByRole('alert');
    expect(alert).toHaveClass('p-2', 'text-xs');
    expect(alert).not.toHaveClass('p-3');
  });

  it('className が追加で付与される', () => {
    render(<Alert className="mb-3">msg</Alert>);

    expect(screen.getByRole('alert')).toHaveClass('mb-3');
  });
});
