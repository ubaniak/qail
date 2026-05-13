// Notification provider — a thin facade over Antd's App.useApp() message
// API so call sites stay close to qail_ui's `toast.success(...)` shape
// instead of importing antd from every page.

import { App } from "antd";

export type Notifier = {
  info: (message: string) => void;
  success: (message: string) => void;
  error: (message: string) => void;
};

// Returns the bound notifier. Must be called inside the antd <App>
// wrapper (set up in main.tsx); App.useApp() returns the in-tree message
// instance so toasts inherit ConfigProvider theme tokens.
export const useNotification = (): { toast: Notifier } => {
  const { message } = App.useApp();
  return {
    toast: {
      info: (m) => void message.info(m),
      success: (m) => void message.success(m),
      error: (m) => void message.error(m),
    },
  };
};
