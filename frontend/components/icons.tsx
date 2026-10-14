type P = { size?: number };

const base = (size: number) => ({
  width: size,
  height: size,
  viewBox: "0 0 24 24",
  fill: "none",
  stroke: "currentColor",
  strokeLinecap: "round" as const,
  strokeLinejoin: "round" as const,
  "aria-hidden": true,
});

export const ChevronLeft = ({ size = 18 }: P) => (
  <svg {...base(size)} strokeWidth={1.6}>
    <path d="M15 5l-7 7 7 7" />
  </svg>
);

export const ChevronRight = ({ size = 18 }: P) => (
  <svg {...base(size)} strokeWidth={1.6}>
    <path d="M9 5l7 7-7 7" />
  </svg>
);

export const Calendar = ({ size = 19 }: P) => (
  <svg {...base(size)} strokeWidth={1.5}>
    <rect x="3.5" y="5" width="17" height="15.5" rx="3" />
    <path d="M3.5 10h17M8 3v4M16 3v4" />
  </svg>
);

export const Plus = ({ size = 18 }: P) => (
  <svg {...base(size)} strokeWidth={2}>
    <path d="M12 5v14M5 12h14" />
  </svg>
);

export const Close = ({ size = 20 }: P) => (
  <svg {...base(size)} strokeWidth={1.6}>
    <path d="M6 6l12 12M18 6L6 18" />
  </svg>
);

export const Moon = ({ size = 14 }: P) => (
  <svg {...base(size)} strokeWidth={1.8}>
    <path d="M20 14.5A8 8 0 0 1 9.5 4a8 8 0 1 0 10.5 10.5z" />
  </svg>
);

export const Check = ({ size = 18 }: P) => (
  <svg {...base(size)} strokeWidth={2}>
    <path d="M5 12.5l4.5 4.5L19 7.5" />
  </svg>
);

export const Copy = ({ size = 18 }: P) => (
  <svg {...base(size)} strokeWidth={1.8}>
    <rect x="8" y="8" width="12" height="12" rx="2.5" />
    <path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2" />
  </svg>
);

export const Alert = ({ size = 18 }: P) => (
  <svg {...base(size)} strokeWidth={1.8}>
    <circle cx="12" cy="12" r="9" />
    <path d="M12 7.5v5.5M12 16.5v.01" />
  </svg>
);
