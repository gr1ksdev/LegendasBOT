import { memo } from 'react';

interface PerfLineProps {
  accent?: boolean;
  className?: string;
}

export const PerfLine = memo(function PerfLine({ accent, className = '' }: PerfLineProps) {
  const color = accent ? 'var(--accent)' : 'var(--border)';

  return (
    <div className={`relative w-full overflow-hidden ${className}`} style={{ height: 18 }} aria-hidden="true">
      <svg
        className="absolute inset-0 w-full h-full"
        viewBox="0 0 1200 18"
        preserveAspectRatio="none"
        fill="none"
      >
        <path
          d="M0 9 Q 15 1, 30 9 T 60 9 T 90 9 T 120 9 T 150 9 T 180 9 T 210 9 T 240 9 T 270 9 T 300 9 T 330 9 T 360 9 T 390 9 T 420 9 T 450 9 T 480 9 T 510 9 T 540 9 T 570 9 T 600 9 T 630 9 T 660 9 T 690 9 T 720 9 T 750 9 T 780 9 T 810 9 T 840 9 T 870 9 T 900 9 T 930 9 T 960 9 T 990 9 T 1020 9 T 1050 9 T 1080 9 T 1110 9 T 1140 9 T 1170 9 T 1200 9"
          stroke={color}
          strokeWidth="1.5"
          strokeLinecap="round"
          opacity="0.4"
        />
      </svg>
    </div>
  );
});
