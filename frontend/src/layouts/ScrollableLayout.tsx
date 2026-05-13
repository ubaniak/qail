// Vertical-scroll container used inside every page so list rows can
// overflow without growing the whole tab pane. The fixed bordered look
// mirrors qail_ui's Paper.

import type { ReactNode } from "react";

export type ScrollableLayoutProps = {
  children: ReactNode;
  className?: string;
};

export const ScrollableLayout = ({
  children,
  className = "",
}: ScrollableLayoutProps) => {
  return (
    <div
      className={`overflow-auto h-full px-2 py-2 ${className}`}
    >
      {children}
    </div>
  );
};
