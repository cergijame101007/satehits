import type { TextareaHTMLAttributes } from 'react';
import { inputClassName } from '@/components/react/ui/inputStyles';
import { cx } from '@/lib/cx';

export interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  hasError?: boolean;
  inputSize?: 'base' | 'sm';
}

export default function Textarea({
  hasError = false,
  inputSize = 'base',
  className,
  ...props
}: TextareaProps) {
  return (
    <textarea
      className={cx(inputClassName(hasError, inputSize), className)}
      {...props}
    />
  );
}
