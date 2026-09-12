import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Button from '@/components/react/ui/Button';

describe('Button', () => {
  it('子要素をラベルとして表示し、既定の type は button になる', () => {
    render(<Button>保存する</Button>);

    const button = screen.getByRole('button', { name: '保存する' });
    expect(button).toHaveAttribute('type', 'button');
  });

  it('type="submit" を渡すとそのまま反映される', () => {
    render(<Button type="submit">送信</Button>);

    expect(screen.getByRole('button', { name: '送信' })).toHaveAttribute('type', 'submit');
  });

  it('クリックで onClick が呼ばれる', async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(<Button onClick={onClick}>押す</Button>);

    await user.click(screen.getByRole('button', { name: '押す' }));

    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('disabled のときはクリックしても onClick が呼ばれない', async () => {
    const user = userEvent.setup();
    const onClick = vi.fn();
    render(
      <Button onClick={onClick} disabled>
        押す
      </Button>,
    );

    const button = screen.getByRole('button', { name: '押す' });
    expect(button).toBeDisabled();
    await user.click(button);

    expect(onClick).not.toHaveBeenCalled();
  });

  it('variant に応じたスタイルが適用される', () => {
    const { rerender } = render(<Button>btn</Button>);
    expect(screen.getByRole('button')).toHaveClass('bg-primary');

    rerender(<Button variant="danger">btn</Button>);
    expect(screen.getByRole('button')).toHaveClass('bg-red-600');

    rerender(<Button variant="ghost">btn</Button>);
    expect(screen.getByRole('button')).toHaveClass('border-gray-300');
  });

  it('size="lg" は全幅、fullWidth は md でも全幅になる', () => {
    const { rerender } = render(<Button size="lg">btn</Button>);
    expect(screen.getByRole('button')).toHaveClass('w-full');

    rerender(<Button size="md">btn</Button>);
    expect(screen.getByRole('button')).not.toHaveClass('w-full');

    rerender(
      <Button size="md" fullWidth>
        btn
      </Button>,
    );
    expect(screen.getByRole('button')).toHaveClass('w-full');
  });
});
