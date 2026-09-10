import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ConfirmModal from '@/components/react/ui/ConfirmModal';

const baseProps = {
  open: true,
  title: '予約を承認',
  description: '山田さんの予約を承認します。',
  confirmLabel: '承認する',
  onConfirm: () => {},
  onCancel: () => {},
};

describe('ConfirmModal', () => {
  it('タイトル・説明・確認ボタン・既定のキャンセルボタンを表示する', () => {
    render(<ConfirmModal {...baseProps} />);

    expect(screen.getByRole('dialog', { name: '予約を承認' })).toBeInTheDocument();
    expect(screen.getByText('山田さんの予約を承認します。')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '承認する' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'キャンセル' })).toBeInTheDocument();
  });

  it('open=false なら描画しない', () => {
    render(<ConfirmModal {...baseProps} open={false} />);

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('確認ボタンで onConfirm、キャンセルで onCancel が呼ばれる', async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    render(<ConfirmModal {...baseProps} onConfirm={onConfirm} onCancel={onCancel} />);

    await user.click(screen.getByRole('button', { name: '承認する' }));
    expect(onConfirm).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole('button', { name: 'キャンセル' }));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it('閉じるボタンでも onCancel が呼ばれる', async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    render(<ConfirmModal {...baseProps} onCancel={onCancel} />);

    await user.click(screen.getByRole('button', { name: '閉じる' }));

    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it('isSubmitting 中は「処理中...」表示でボタンが無効になる', async () => {
    const user = userEvent.setup();
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    render(
      <ConfirmModal {...baseProps} isSubmitting onConfirm={onConfirm} onCancel={onCancel} />,
    );

    const confirm = screen.getByRole('button', { name: '処理中...' });
    expect(confirm).toBeDisabled();
    expect(screen.getByRole('button', { name: 'キャンセル' })).toBeDisabled();
    expect(screen.getByRole('button', { name: '閉じる' })).toBeDisabled();

    await user.click(confirm);
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it('cancelLabel と confirmVariant を上書きできる', () => {
    render(<ConfirmModal {...baseProps} cancelLabel="やめる" confirmVariant="danger" />);

    expect(screen.getByRole('button', { name: 'やめる' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '承認する' })).toHaveClass('bg-red-600');
  });

  it('children を説明文の下に表示する', () => {
    render(
      <ConfirmModal {...baseProps}>
        <textarea aria-label="拒否理由" />
      </ConfirmModal>,
    );

    expect(screen.getByRole('textbox', { name: '拒否理由' })).toBeInTheDocument();
  });
});
