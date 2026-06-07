import type { filter } from '../../wailsjs/go/models';
import { StatCard } from './StatCard';

type Stats = filter.Stats;

interface DashboardProps {
  stats: Stats | null;
  uptime: string;
  dbInfo: Record<string, any>;
  cacheInfo: Record<string, any>;
  protected_: boolean;
  onToggle: () => void;
}

export function Dashboard({ stats, uptime, dbInfo, cacheInfo, protected_, onToggle }: DashboardProps) {
  const blockedRate = stats && (stats.blocked + stats.allowed) > 0
    ? ((stats.blocked / (stats.blocked + stats.allowed)) * 100).toFixed(1)
    : '0.0';

  return (
    <>
      <div className="toggle-section">
        <button
          className={`toggle-btn ${protected_ ? 'on' : 'off'}`}
          onClick={onToggle}
        >
          <div className="toggle-knob" />
          <span className="toggle-label">
            {protected_ ? 'Ochrona aktywna' : 'Ochrona wyłączona'}
          </span>
        </button>
      </div>

      <div className="stats-grid">
        <StatCard label="Zablokowane" value={stats?.blocked ?? 0} color="#ef4444" />
        <StatCard label="Przepuszczone" value={stats?.allowed ?? 0} color="#22c55e" />
        <StatCard label="Współczynnik blokad" value={blockedRate} unit="%" color="#f59e0b" />
        <StatCard label="Zakresy IP" value={dbInfo['ranges'] ?? 0} color="#3b82f6" />
        <StatCard label="Uptime" value={uptime} color="#8b5cf6" />
        <StatCard label="Upuszczone" value={stats?.dropped ?? 0} color="#64748b" />
        <StatCard
          label="Cache"
          value={`${(cacheInfo['entries'] ?? 0).toLocaleString()} / ${(cacheInfo['max'] ?? 65536).toLocaleString()}`}
          color="#64748b"
        />
      </div>
    </>
  );
}
