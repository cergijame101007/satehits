import type { ReactNode } from 'react';
import Button, { type ButtonVariant } from '@/components/react/ui/Button';
import Modal from '@/components/react/ui/Modal';
import { cx } from '@/lib/cx';

export interface ConfirmModalProps {
  open: boolean;
  title: string;
  description: string;
  confirmLabel: string;
  confirmVariant?: ButtonVariant;
  cancelLabel?: string;
  isSubmitting?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  children?: ReactNode;
  bodyClassName?: string;
}

export default function ConfirmModal({
  open,
  title,
  description,
  confirmLabel,
  confirmVariant = 'primary',
  cancelLabel = 'キャンセル',
  isSubmitting = false,
  onConfirm,
  onCancel,
  children,
  bodyClassName,
}: ConfirmModalProps) {
  return (
    <Modal
      open={open}
      title={title}
      onClose={onCancel}
      closeDisabled={isSubmitting}
      footer={
        <div className={cx('flex justify-end gap-3 p-5', children ? '' : 'pt-0')}>
          <Button variant="ghost" size="md" onClick={onCancel} disabled={isSubmitting}>
            {cancelLabel}
          </Button>
          <Button
            variant={confirmVariant}
            size="md"
            onClick={onConfirm}
            disabled={isSubmitting}
          >
            {isSubmitting ? '処理中...' : confirmLabel}
          </Button>
        </div>
      }
    >
      <p className={cx('text-sm text-gray-600', bodyClassName)}>{description}</p>
      {children}
    </Modal>
  );
}
