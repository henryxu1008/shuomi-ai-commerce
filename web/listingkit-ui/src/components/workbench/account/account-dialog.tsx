"use client";
import { useEffect, useId, useRef, type RefObject } from "react";
import { Button } from "@/components/ui/button";
import styles from "./roles.module.css";

export function AccountDialog({ title, drawer = false, className, onClose, children, returnFocusRef }: {
  title: string;
  drawer?: boolean;
  className?: string;
  onClose: () => void;
  children: React.ReactNode;
  returnFocusRef?: RefObject<HTMLElement | null>;
}) {
  const ref = useRef<HTMLDialogElement>(null), close = useRef<HTMLButtonElement>(null), id = useId();
  useEffect(() => {
    const element = ref.current, previous = returnFocusRef?.current ?? document.activeElement;
    element?.showModal(); close.current?.focus();
    return () => { element?.close(); if (previous instanceof HTMLElement && previous.isConnected) previous.focus(); };
  }, [returnFocusRef]);
  return <dialog ref={ref} className={[drawer ? styles.drawer : styles.modal, className].filter(Boolean).join(" ")} aria-labelledby={id} onCancel={e => { e.preventDefault(); onClose(); }}><header><h2 id={id}>{title}</h2><Button ref={close} variant="ghost" aria-label={`关闭${title}`} onClick={onClose}>×</Button></header>{children}</dialog>;
}
