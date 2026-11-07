"use client";
import { useLayoutEffect, useRef, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import styles from "./resources.module.css";
export function ResourceDialog({
  title,
  onClose,
  locked = false,
  children,
  className,
}: {
  title: string;
  onClose: () => void;
  locked?: boolean;
  children: ReactNode;
  className?: string;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useLayoutEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    const trigger = document.activeElement;
    dialog.showModal();
    return () => {
      const focused = document.activeElement;
      const restoreFocus = focused === document.body || dialog.contains(focused);
      dialog.close();
      if (restoreFocus && trigger instanceof HTMLElement && trigger.isConnected && document.activeElement !== trigger) {
        trigger.focus({ preventScroll: true });
      }
    };
  }, []);
  return (
    <dialog
      ref={ref}
      className={[styles.resourceDialog,className].filter(Boolean).join(" ")}
      aria-label={title}
      onCancel={(event) => {
        event.preventDefault();
        if (!locked) onClose();
      }}
    >
      <div className={styles.dialogHeading}>
        <h2>{title}</h2>
        <Button
          variant="outline"
          disabled={locked}
          onClick={onClose}
          aria-label="关闭"
        >
          ×
        </Button>
      </div>
      {children}
    </dialog>
  );
}
