import type { AnchorHTMLAttributes, ReactNode } from 'react';
import { getButtonClassName } from '@/lib/ui/buttonStyles';
import type { ButtonSize, ButtonVariant } from '@/lib/ui/types';

export interface ButtonLinkProps extends Omit<AnchorHTMLAttributes<HTMLAnchorElement>, 'className'> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  fullWidth?: boolean;
  children: ReactNode;
}

export default function ButtonLink({
  variant = 'primary',
  size = 'md',
  fullWidth = false,
  children,
  ...props
}: ButtonLinkProps) {
  return (
    <a className={getButtonClassName(variant, size, fullWidth)} {...props}>
      {children}
    </a>
  );
}
