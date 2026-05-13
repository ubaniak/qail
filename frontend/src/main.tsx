// Provider stack:
//   ConfigProvider sets Antd's dark algorithm + brand tokens
//   App (antd) supplies the message/notification/modal context used by
//     useNotification() + Popconfirm/Modal hooks
//   ProgressProvider wires Wails workspace:* events to the log drawer
//   QailServiceProvider exposes the read+write surface to every page
//
// The ordering matters: notification needs <App>; qailservice needs
// notification + progress.

import { App as AntApp, ConfigProvider, theme } from "antd";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import App from "./App";
import { ProgressProvider } from "./providers/progress";
import { QailServiceProvider } from "./providers/qailservice";
import "./index.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ConfigProvider
      theme={{
        algorithm: theme.darkAlgorithm,
        token: {
          colorPrimary: "#10b981", // emerald-500
          colorInfo: "#10b981",
          colorBgBase: "#09090b", // zinc-950
          colorBgContainer: "#18181b", // zinc-900
          colorBgElevated: "#27272a", // zinc-800
          borderRadius: 8,
          fontFamily:
            '-apple-system, BlinkMacSystemFont, "Inter", system-ui, "Segoe UI", Roboto, sans-serif',
        },
        components: {
          Tabs: {
            inkBarColor: "#34d399",
            itemSelectedColor: "#34d399",
            itemHoverColor: "#a7f3d0",
          },
        },
      }}
    >
      <AntApp>
        <ProgressProvider>
          <QailServiceProvider>
            <App />
          </QailServiceProvider>
        </ProgressProvider>
      </AntApp>
    </ConfigProvider>
  </StrictMode>
);
