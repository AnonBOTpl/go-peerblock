import { useState, useMemo, useCallback } from 'react';
import { SourceDialog } from './SourceDialog';
import { LookupBlockSource } from '../../wailsjs/go/main/App';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js';
import { Line } from 'react-chartjs-2';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
);

export interface Sample {
  time: number; // Date.now()
  blockedPPS: number;
  allowedPPS: number;
}

export interface BlockedEntry {
  id: number;
  timestamp: number;
  srcIP: string;
  dstIP: string;
  proto: string;
}

interface ChartsViewProps {
  history: Sample[];
  blockedEntries: BlockedEntry[];
}

type Range = 5 | 10 | 30;

const RANGES: { value: Range; label: string }[] = [
  { value: 5, label: '5 min' },
  { value: 10, label: '10 min' },
  { value: 30, label: '30 min' },
];

function formatTime(ts: number): string {
  const d = new Date(ts);
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

function formatBlockTime(ts: number): string {
  const d = new Date(ts);
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}:${String(d.getSeconds()).padStart(2, '0')}`;
}

export function ChartsView({ history, blockedEntries }: ChartsViewProps) {
  const [timeRange, setTimeRange] = useState<Range>(5);
  const [dialogIP, setDialogIP] = useState<string | null>(null);
  const [dialogSources, setDialogSources] = useState<string[]>([]);
  const [loadingSource, setLoadingSource] = useState(false);

  // Filter history by selected time range
  const filtered = useMemo(() => {
    const cutoff = Date.now() - timeRange * 60 * 1000;
    return history.filter(h => h.time >= cutoff);
  }, [history, timeRange]);

  const data = useMemo(() => ({
    labels: filtered.map(h => formatTime(h.time)),
    datasets: [
      {
        label: 'Blokowane',
        data: filtered.map(h => h.blockedPPS),
        borderColor: '#ef4444',
        backgroundColor: 'rgba(239, 68, 68, 0.08)',
        fill: true,
        tension: 0.3,
        pointRadius: 0,
        pointHitRadius: 8,
        borderWidth: 2,
      },
      {
        label: 'Przepuszczone',
        data: filtered.map(h => h.allowedPPS),
        borderColor: '#22c55e',
        backgroundColor: 'rgba(34, 197, 94, 0.08)',
        fill: true,
        tension: 0.3,
        pointRadius: 0,
        pointHitRadius: 8,
        borderWidth: 2,
      },
    ],
  }), [filtered]);

  const handleBlockClick = useCallback(async (entry: BlockedEntry) => {
    setDialogIP(entry.dstIP);
    setDialogSources([]);
    setLoadingSource(true);
    try {
      const sources = await LookupBlockSource(entry.dstIP);
      setDialogSources(sources || []);
    } catch {
      setDialogSources([]);
    }
    setLoadingSource(false);
  }, []);

  const handleCloseDialog = useCallback(() => {
    setDialogIP(null);
    setDialogSources([]);
  }, []);

  const options = {
    responsive: true,
    maintainAspectRatio: false,
    animation: false as const,
    interaction: {
      mode: 'index' as const,
      intersect: false,
    },
    scales: {
      x: {
        display: true,
        grid: { display: false },
        ticks: {
          color: '#64748b',
          font: { size: 10 },
          maxTicksLimit: 12,
        },
      },
      y: {
        display: true,
        beginAtZero: true,
        grid: {
          color: 'rgba(255,255,255,0.04)',
        },
        ticks: {
          color: '#64748b',
          font: { size: 10 },
          callback: (value: any) => `${value}/s`,
        },
        title: {
          display: true,
          text: 'Pakiety / s',
          color: '#94a3b8',
          font: { size: 11 },
        },
      },
    },
    plugins: {
      legend: {
        position: 'top' as const,
        align: 'end' as const,
        labels: {
          boxWidth: 12,
          boxHeight: 12,
          padding: 16,
          color: '#f1f5f9',
          font: { size: 12, weight: 700 },
          usePointStyle: true,
          pointStyle: 'circle' as const,
        },
      },
      tooltip: {
        backgroundColor: '#1e293b',
        titleColor: '#94a3b8',
        titleFont: { size: 11 },
        bodyFont: { size: 13 },
        borderColor: '#334155',
        borderWidth: 1,
        padding: 10,
        callbacks: {
          title: (items: any[]) => items[0]?.label || '',
          label: (ctx: any) => {
            const val = ctx.parsed.y;
            return `${ctx.dataset.label}: ${val.toFixed(1)}/s`;
          },
        },
      },
    },
  };

  return (
    <div className="charts-view">
      <div className="charts-header">
        <h2>Wykres ruchu</h2>
        <div className="chart-range-buttons">
          {RANGES.map(r => (
            <button
              key={r.value}
              className={`chart-range-btn ${timeRange === r.value ? 'active' : ''}`}
              onClick={() => setTimeRange(r.value)}
            >
              {r.label}
            </button>
          ))}
        </div>
      </div>

      {filtered.length < 2 ? (
        <div className="charts-empty">
          <div className="charts-empty-icon">📈</div>
          <div className="charts-empty-text">Zbieranie danych...</div>
          <div className="charts-empty-hint">Wykres pojawi się za chwilę, gdy zgromadzimy wystarczająco próbek.</div>
        </div>
      ) : (
        <div className="chart-container">
          <Line data={data} options={options} />
        </div>
      )}

      {/* Blocked IPs list */}
      <div className="blocked-list">
        <div className="blocked-list-header">
          <h3>Ostatnie blokady</h3>
          <span className="blocked-list-count">{blockedEntries.length}</span>
        </div>
        {blockedEntries.length === 0 ? (
          <div className="blocked-list-empty">Brak zablokowanych pakietów. Włącz ochronę aby zobaczyć listę.</div>
        ) : (
          <div className="blocked-list-entries">
            {blockedEntries.map((entry) => (
                <div
                  key={entry.id}
                  className="blocked-entry"
                  onClick={() => handleBlockClick(entry)}
                  title="Kliknij aby sprawdzić źródło blokady"
                >
                  <span className="blocked-time">{formatBlockTime(entry.timestamp)}</span>
                  <span className="blocked-ips">
                    <span className="blocked-src">{entry.srcIP}</span>
                    <span className="blocked-arrow">→</span>
                    <span className="blocked-dst">{entry.dstIP}</span>
                  </span>
                  <span className="blocked-proto">{entry.proto}</span>
                  <span className="blocked-lookup-icon">🔍</span>
                </div>
              ))}
          </div>
        )}
      </div>

      {/* Source dialog */}
      {dialogIP && (
        <SourceDialog
          ip={dialogIP}
          sources={loadingSource ? ['Szukanie źródeł...'] : dialogSources}
          onClose={handleCloseDialog}
        />
      )}
    </div>
  );
}
