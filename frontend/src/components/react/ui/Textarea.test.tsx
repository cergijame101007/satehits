import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Textarea from '@/components/react/ui/Textarea';

describe('Textarea', () => {
  it('textarea として描画され、入力で onChange が呼ばれる', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<Textarea aria-label="備考" onChange={onChange} />);

    await user.type(screen.getByRole('textbox', { name: '備考' }), 'abc');

    expect(onChange).toHaveBeenCalledTimes(3);
  });

  it('hasError でエラー用の枠線スタイルになる', () => {
    const { rerender } = render(<Textarea aria-label="備考" />);
    expect(screen.getByRole('textbox')).toHaveClass('border-gray-200');

    rerender(<Textarea aria-label="備考" hasError />);
    expect(screen.getByRole('textbox')).toHaveClass('border-red-400');
  });

  it('inputSize="sm" で小さい文字サイズになる', () => {
    render(<Textarea aria-label="備考" inputSize="sm" />);

    expect(screen.getByRole('textbox')).toHaveClass('text-sm');
  });

  it('rows / placeholder / className がそのまま渡される', () => {
    render(<Textarea aria-label="備考" rows={4} placeholder="例: 定員超過" className="extra" />);

    const textarea = screen.getByRole('textbox');
    expect(textarea).toHaveAttribute('rows', '4');
    expect(textarea).toHaveAttribute('placeholder', '例: 定員超過');
    expect(textarea).toHaveClass('extra');
  });
});
