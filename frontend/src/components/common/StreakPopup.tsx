import { useState, useEffect, useCallback, type MouseEvent } from 'react';

const MOTIVATIONAL_QUOTES: string[] = [
  'You came back. Suspicious… but impressive.',
  "Streak increased. Don't mess it up tomorrow.",
  'Oh, so you *can* be consistent.',
  'You again? Fine. Keep going.',
  "Look who didn't quit today.",
  "Streak saved. Barely. Don't test your luck.",
  "You're building a streak… or a personality?",
  'Another day? Who are you trying to impress?',
  "You showed up. We're watching.",
  'Good. Now do it again tomorrow.',
  "You didn't break the streak. Shocking.",
  'Consistency? From you? Interesting.',
  "You're on a roll. Don't trip now.",
  "We expected you to skip. You didn't.",
  'Streak +1. Pressure +10.',
  "You're getting harder to ignore.",
  'You did it today. No excuses tomorrow.',
  "This is getting serious… don't ruin it.",
  'You came back again. This might be a habit.',
  "Don't celebrate too early. Tomorrow is waiting.",
  "You're still here. That's dangerous.",
  'Keep this up and you might actually succeed.',
  "Not bad. Don't get comfortable.",
  'You survived today. Repeat.',
  "You're proving us wrong. Continue.",
];

export interface StreakPopupProps {
  streak: number;
  onClose: () => void;
}

function StreakPopup({ streak, onClose }: StreakPopupProps) {
  const [isVisible, setIsVisible] = useState(false);
  const [quote] = useState(
    () => MOTIVATIONAL_QUOTES[Math.floor(Math.random() * MOTIVATIONAL_QUOTES.length)]
  );

  const handleClose = useCallback(() => {
    setIsVisible(false);
    setTimeout(() => onClose(), 300);
  }, [onClose]);

  useEffect(() => {
    const timer = setTimeout(() => setIsVisible(true), 50);
    const closeTimer = setTimeout(() => {
      handleClose();
    }, 4000);
    return () => {
      clearTimeout(timer);
      clearTimeout(closeTimer);
    };
  }, [streak, handleClose]);

  const getStreakMessage = (count: number): string => {
    if (count === 1) return '1 Day Streak!';
    if (count === 2) return '2 Day Streak!';
    if (count === 3) return '3 Day Streak!';
    if (count === 7) return '1 Week Streak!';
    if (count === 14) return '2 Week Streak!';
    if (count === 21) return '3 Week Streak!';
    if (count === 30) return '1 Month Streak!';
    if (count === 60) return '2 Month Streak!';
    if (count === 90) return '3 Month Streak!';
    if (count === 100) return '100 Day Streak!';
    if (count === 365) return '1 Year Streak!';
    return `${count} Day Streak!`;
  };

  const getStreakEmoji = (count: number): string => {
    if (count >= 365) return '🏆';
    if (count >= 100) return '💯';
    if (count >= 30) return '🌟';
    if (count >= 7) return '🔥';
    return '⚡';
  };

  return (
    <div
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        backgroundColor: 'rgba(0, 0, 0, 0.7)',
        backdropFilter: 'blur(4px)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 9999,
        opacity: isVisible ? 1 : 0,
        transition: 'opacity 0.3s ease',
      }}
      onClick={handleClose}
    >
      <div
        style={{
          background: 'var(--bg-secondary)',
          borderRadius: '20px',
          padding: '40px 50px',
          textAlign: 'center',
          maxWidth: '450px',
          width: '90%',
          border: '2px solid var(--accent-secondary-strong)',
          boxShadow:
            '0 20px 60px var(--accent-secondary-medium), 0 0 100px var(--accent-secondary-soft)',
          transform: isVisible ? 'scale(1) translateY(0)' : 'scale(0.8) translateY(20px)',
          transition: 'transform 0.3s cubic-bezier(0.34, 1.56, 0.64, 1)',
        }}
        onClick={(e: MouseEvent<HTMLDivElement>) => e.stopPropagation()}
      >
        <div
          style={{
            fontSize: '80px',
            marginBottom: '20px',
            animation: 'pulse 1s ease-in-out infinite',
          }}
        >
          {getStreakEmoji(streak)}
        </div>
        <h2
          style={{
            fontSize: '2.5rem',
            fontWeight: 'bold',
            margin: '0 0 15px 0',
            background: 'linear-gradient(135deg, var(--amber-500), var(--amber-400))',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
            backgroundClip: 'text',
          }}
        >
          {getStreakMessage(streak)}
        </h2>
        <p
          style={{
            fontSize: '1.1rem',
            color: 'var(--text-secondary)',
            fontStyle: 'italic',
            margin: '0 0 30px 0',
            lineHeight: '1.5',
          }}
        >
          "{quote}"
        </p>
        <button
          onClick={handleClose}
          style={{
            background: 'linear-gradient(135deg, var(--amber-500), var(--amber-400))',
            border: 'none',
            padding: '12px 35px',
            borderRadius: '25px',
            color: '#fff',
            fontSize: '1rem',
            fontWeight: '600',
            cursor: 'pointer',
            transition: 'transform 0.2s, box-shadow 0.2s',
            boxShadow: '0 4px 15px var(--accent-secondary-strong)',
          }}
          onMouseEnter={(e) => {
            (e.currentTarget as HTMLButtonElement).style.transform = 'scale(1.05)';
            (e.currentTarget as HTMLButtonElement).style.boxShadow =
              '0 6px 20px rgba(245, 158, 11, 0.5)';
          }}
          onMouseLeave={(e) => {
            (e.currentTarget as HTMLButtonElement).style.transform = 'scale(1)';
            (e.currentTarget as HTMLButtonElement).style.boxShadow =
              '0 4px 15px var(--accent-secondary-strong)';
          }}
        >
          Keep Going →
        </button>
      </div>
      <style>{`
        @keyframes pulse {
          0%, 100% { transform: scale(1); }
          50% { transform: scale(1.1); }
        }
      `}</style>
    </div>
  );
}

export default StreakPopup;
