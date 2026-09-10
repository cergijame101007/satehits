import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import Card from '@/components/react/ui/Card';

describe('Card', () => {
  it('子要素を枠付きで表示する', () => {
    render(
      <Card>
        <p>内容</p>
      </Card>,
    );

    const content = screen.getByText('内容');
    expect(content.parentElement).toHaveClass('rounded-xl', 'border', 'p-5');
  });

  it('compact 指定でパディングが小さくなる', () => {
    render(
      <Card compact>
        <p>内容</p>
      </Card>,
    );

    const card = screen.getByText('内容').parentElement;
    expect(card).toHaveClass('p-4');
    expect(card).not.toHaveClass('p-5');
  });

  it('className が追加で付与される', () => {
    render(
      <Card className="lg:col-span-2">
        <p>内容</p>
      </Card>,
    );

    expect(screen.getByText('内容').parentElement).toHaveClass('lg:col-span-2');
  });
});
