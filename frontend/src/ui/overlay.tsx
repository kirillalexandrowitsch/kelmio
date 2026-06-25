import {
  type MouseEvent as ReactMouseEvent,
  type ReactNode,
  useEffect,
  useRef,
} from "react";
import { createPortal } from "react-dom";

type OverlayPlacement = "center" | "end";

type OverlayProps = {
  open: boolean;
  onClose: () => void;
  label: string;
  placement?: OverlayPlacement;
  panelClassName: string;
  className?: string;
  children: ReactNode;
};

function Overlay({
  open,
  onClose,
  label,
  placement = "center",
  panelClassName,
  className,
  children,
}: OverlayProps) {
  const panelRef = useRef<HTMLDivElement>(null);
  const previouslyFocused = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (!open) {
      return;
    }

    previouslyFocused.current = document.activeElement as HTMLElement | null;
    panelRef.current?.focus();

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        event.stopPropagation();
        onClose();
      }
    }

    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      previouslyFocused.current?.focus?.();
    };
  }, [open, onClose]);

  if (!open) {
    return null;
  }

  function onBackdropMouseDown(event: ReactMouseEvent<HTMLDivElement>) {
    if (event.target === event.currentTarget) {
      onClose();
    }
  }

  return createPortal(
    <div
      className={["kl-overlay", `kl-overlay--${placement}`, className]
        .filter(Boolean)
        .join(" ")}
      onMouseDown={onBackdropMouseDown}
    >
      <div
        ref={panelRef}
        className={panelClassName}
        role="dialog"
        aria-modal="true"
        aria-label={label}
        tabIndex={-1}
      >
        {children}
      </div>
    </div>,
    document.body,
  );
}

type OverlayVariantProps = Omit<OverlayProps, "placement" | "panelClassName"> & {
  panelClassName?: string;
};

export function Modal({ panelClassName, ...rest }: OverlayVariantProps) {
  return (
    <Overlay
      {...rest}
      placement="center"
      panelClassName={["kl-modal", panelClassName].filter(Boolean).join(" ")}
    />
  );
}

export function SlideOver({ panelClassName, ...rest }: OverlayVariantProps) {
  return (
    <Overlay
      {...rest}
      placement="end"
      panelClassName={["kl-slideover", panelClassName].filter(Boolean).join(" ")}
    />
  );
}
