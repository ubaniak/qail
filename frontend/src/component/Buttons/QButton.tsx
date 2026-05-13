// Branded button with three semantic variants. Wraps Antd Button so
// callers can stay agnostic about colors / hover states.

import { Button } from "antd";
import type { ButtonProps } from "antd";
import { forwardRef } from "react";

export type QButtonVariant = "accent" | "destructive" | "cancel" | "neutral";

export interface QButtonProps extends Omit<ButtonProps, "type" | "danger" | "variant"> {
  variant?: QButtonVariant;
}

export const QButton = forwardRef<HTMLButtonElement, QButtonProps>(
  ({ variant = "accent", className = "", children, ...props }, ref) => {
    let type: ButtonProps["type"] = "default";
    let danger = false;

    switch (variant) {
      case "accent":
        type = "primary";
        break;
      case "destructive":
        type = "primary";
        danger = true;
        break;
      case "cancel":
        type = "default";
        break;
      case "neutral":
        type = "default";
        break;
    }

    return (
      <Button
        ref={ref}
        type={type}
        danger={danger}
        className={`!font-semibold ${className}`}
        {...props}
      >
        {children}
      </Button>
    );
  }
);

QButton.displayName = "QButton";
