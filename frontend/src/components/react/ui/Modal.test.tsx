import { useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Modal from '@/components/react/ui/Modal';

describe('Modal', () => {
  it('open=false のときは何も描画しない', () => {
    render(
      <Modal open={false} title="タイトル" onClose={() => {}}>
        <p>本文</p>
      </Modal>,
    );

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('open=true でタイトル・本文・フッターを表示する', () => {
    render(
      <Modal open title="取引先を編集" onClose={() => {}} footer={<button type="button">保存</button>}>
        <p>本文</p>
      </Modal>,
    );

    const dialog = screen.getByRole('dialog', { name: '取引先を編集' });
    expect(dialog).toHaveAttribute('aria-modal', 'true');
    expect(screen.getByText('本文')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '保存' })).toBeInTheDocument();
  });

  it('閉じるボタンのクリックで onClose が呼ばれる', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(
      <Modal open title="タイトル" onClose={onClose}>
        <p>本文</p>
      </Modal>,
    );

    await user.click(screen.getByRole('button', { name: '閉じる' }));

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('Escape キーで onClose が呼ばれる', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(
      <Modal open title="タイトル" onClose={onClose}>
        <p>本文</p>
      </Modal>,
    );

    await user.keyboard('{Escape}');

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('背景クリックで onClose が呼ばれる', () => {
    const onClose = vi.fn();
    const { container } = render(
      <Modal open title="タイトル" onClose={onClose}>
        <p>本文</p>
      </Modal>,
    );

    const backdrop = container.querySelector('[aria-hidden="true"]');
    expect(backdrop).not.toBeNull();
    fireEvent.click(backdrop!);

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('closeDisabled のときは閉じる操作がすべて無効になる', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const { container } = render(
      <Modal open title="タイトル" onClose={onClose} closeDisabled>
        <p>本文</p>
      </Modal>,
    );

    expect(screen.getByRole('button', { name: '閉じる' })).toBeDisabled();
    await user.keyboard('{Escape}');
    fireEvent.click(container.querySelector('[aria-hidden="true"]')!);

    expect(onClose).not.toHaveBeenCalled();
  });

  it('開いたとき最初のフォーカス可能要素にフォーカスし、body のスクロールを止める', () => {
    const { unmount } = render(
      <Modal open title="タイトル" onClose={() => {}}>
        <input aria-label="名前" />
      </Modal>,
    );

    // ヘッダーの閉じるボタンが最初のフォーカス可能要素
    expect(screen.getByRole('button', { name: '閉じる' })).toHaveFocus();
    expect(document.body.style.overflow).toBe('hidden');

    unmount();
    expect(document.body.style.overflow).toBe('');
  });

  it('Tab でフォーカスがダイアログ内を循環する', async () => {
    const user = userEvent.setup();
    render(
      <Modal open title="タイトル" onClose={() => {}} footer={<button type="button">最後</button>}>
        <input aria-label="名前" />
      </Modal>,
    );

    const last = screen.getByRole('button', { name: '最後' });
    last.focus();
    await user.tab();

    expect(screen.getByRole('button', { name: '閉じる' })).toHaveFocus();

    await user.tab({ shift: true });
    expect(last).toHaveFocus();
  });

  it('onClose の参照が毎レンダー変わっても、入力中のフォーカスを奪わない', async () => {
    const user = userEvent.setup();

    function Wrapper() {
      const [value, setValue] = useState('');
      // 親の再レンダーごとに新しい onClose を渡す（ReservationTable 等と同じ状況）
      return (
        <Modal open title="タイトル" onClose={() => {}}>
          <textarea aria-label="理由" value={value} onChange={(e) => setValue(e.target.value)} />
        </Modal>
      );
    }

    render(<Wrapper />);
    const textarea = screen.getByRole('textbox', { name: '理由' });
    await user.click(textarea);
    await user.type(textarea, '定員超過のため');

    expect(textarea).toHaveValue('定員超過のため');
    expect(textarea).toHaveFocus();
  });

  it('再レンダー後も最新の onClose が Escape で呼ばれる', async () => {
    const user = userEvent.setup();
    const first = vi.fn();
    const second = vi.fn();
    const { rerender } = render(
      <Modal open title="タイトル" onClose={first}>
        <p>本文</p>
      </Modal>,
    );

    rerender(
      <Modal open title="タイトル" onClose={second}>
        <p>本文</p>
      </Modal>,
    );
    await user.keyboard('{Escape}');

    expect(first).not.toHaveBeenCalled();
    expect(second).toHaveBeenCalledTimes(1);
  });

  it('size="lg" で幅の大きいダイアログになる', () => {
    render(
      <Modal open title="タイトル" onClose={() => {}} size="lg">
        <p>本文</p>
      </Modal>,
    );

    expect(screen.getByRole('dialog')).toHaveClass('max-w-lg');
  });
});
