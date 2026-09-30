import { useState, useEffect } from 'react';

export interface SolveTimerProps {
  startTime: string;
  endTime: string;
}

interface TimeLeft {
  status: 'upcoming' | 'active' | 'ended';
  time: number;
}

function SolveTimer({ startTime, endTime }: SolveTimerProps) {
  const [timeLeft, setTimeLeft] = useState<TimeLeft>(() => {
    const now = new Date().getTime();
    const end = new Date(endTime).getTime();
    const start = new Date(startTime).getTime();
    if (now < start) return { status: 'upcoming', time: start - now };
    if (now > end) return { status: 'ended', time: 0 };
    return { status: 'active', time: end - now };
  });

  useEffect(() => {
    const calculateTimeLeft = (): TimeLeft => {
      const now = new Date().getTime();
      const end = new Date(endTime).getTime();
      const start = new Date(startTime).getTime();

      if (now < start) return { status: 'upcoming', time: start - now };
      if (now > end) return { status: 'ended', time: 0 };
      return { status: 'active', time: end - now };
    };

    const timer = setInterval(() => setTimeLeft(calculateTimeLeft()), 1000);
    return () => clearInterval(timer);
  }, [startTime, endTime]);

  const formatTime = (ms: number): string => {
    if (ms < 0) ms = 0;
    const totalSeconds = Math.floor(ms / 1000);
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  };

  if (!timeLeft) return null;

  const getTimeColor = (): string => {
    if (timeLeft.status === 'ended') return 'text-accent-danger';
    if (timeLeft.status === 'upcoming') return 'text-text-muted';
    if (timeLeft.time < 5 * 60 * 1000) return 'text-accent-danger';
    if (timeLeft.time < 15 * 60 * 1000) return 'text-accent-warning';
    return 'text-accent-success';
  };

  return (
    <div className={`flex items-center gap-2 ${getTimeColor()}`}>
      {timeLeft.status === 'upcoming' && <span className="text-xs">Starts in: </span>}
      {timeLeft.status === 'active' && (
        <svg
          width="16"
          height="16"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
        >
          <circle cx="12" cy="12" r="10"></circle>
          <polyline points="12 6 12 12 16 14"></polyline>
        </svg>
      )}
      <span className="font-mono font-semibold">
        {timeLeft.status === 'ended' ? 'Ended' : formatTime(timeLeft.time)}
      </span>
    </div>
  );
}

export default SolveTimer;
