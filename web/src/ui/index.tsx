import type { ButtonHTMLAttributes, HTMLAttributes, ReactNode } from "react";

function cn(...values: (string | false | undefined)[]) {
  return values.filter(Boolean).join(" ");
}

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "outline" | "ghost" | "danger";
  size?: "sm" | "md";
  loading?: boolean;
};

export function Button({ variant = "primary", size = "md", loading, className, children, disabled, type = "button", ...props }: ButtonProps) {
  const tones = {
    primary: "bg-primary text-primary-fg hover:bg-primary-hover",
    outline: "border border-border bg-surface text-fg hover:bg-surface-hover",
    ghost: "text-fg-muted hover:bg-surface-hover hover:text-fg",
    danger: "bg-danger text-danger-fg hover:brightness-110",
  };
  return <button type={type} aria-busy={loading || undefined} disabled={disabled || loading} className={cn("inline-flex items-center justify-center gap-2 rounded-ui font-medium transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50 disabled:cursor-not-allowed", size === "sm" ? "min-h-9 px-3 text-sm" : "min-h-11 px-4 text-sm", tones[variant], className)} {...props}>{children}</button>;
}

export function Card({ className, children, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("rounded-ui border border-border-subtle bg-surface p-5", className)} {...props}>{children}</div>;
}

export function Badge({ tone = "neutral", children }: { tone?: "neutral" | "primary" | "success" | "warning" | "danger"; children: ReactNode }) {
  const tones = { neutral: "bg-surface-muted text-fg-muted", primary: "bg-primary-soft text-primary-text", success: "bg-success-soft text-success-text", warning: "bg-warning-soft text-warning-text", danger: "bg-danger-soft text-danger-text" };
  return <span className={cn("inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold", tones[tone])}>{children}</span>;
}

export function Metric({ label, value, icon }: { label: string; value: ReactNode; icon: ReactNode }) {
  return <Card className="flex items-start justify-between gap-3"><div><p className="text-sm text-fg-muted">{label}</p><p className="mt-3 text-3xl font-semibold tracking-tight text-fg tabular">{value}</p></div><div className="rounded-ui bg-primary-soft p-2.5 text-primary-text">{icon}</div></Card>;
}
