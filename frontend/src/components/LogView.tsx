import { useState, useEffect, useRef } from 'react';
import type { logger } from '../../wailsjs/go/models';

type LogEntry = logger.LogEntry;

interface LogViewProps {
  logs: LogEntry[];
  onClear: () => void;
}

export function LogView({ logs, onClear }: LogViewProps) {
  const logEndRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  // Poziomy: 0=DEBUG, 1=INFO, 2=WARN, 3=ERROR
  const [levelFilter, setLevelFilter] = useState<string>('SYSTEM');

  const filteredLogs = logs.filter(e => {
    if (levelFilter === 'ALL') return true;          // wszystko, włączając DEBUG
    if (levelFilter === 'SYSTEM') return e.level >= 1 && !e.message.startsWith('BLOCK'); // system INFO+ (bez BLOCK)
    if (levelFilter === 'BLOCKED') return e.message.startsWith('BLOCK');
    if (levelFilter === 'ERROR') return e.level >= 3;
    return true;
  });

  useEffect(() => {
    if (autoScroll && logEndRef.current) {
      logEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [filteredLogs.length, autoScroll]);

  const getLevelClass = (level: number, msg: string) => {
    if (msg.startsWith('BLOCK')) return 'log-blocked';
    switch (level) {
      case 0: return 'log-debug';
      case 1: return 'log-info';
      case 2: return 'log-warn';
      case 3: return 'log-error';
      default: return '';
    }
  };

  const getLevelLabel = (level: number) => {
    switch (level) {
      case 0: return 'DBG';
      case 1: return 'INF';
      case 2: return 'WRN';
      case 3: return 'ERR';
      default: return '?';
    }
  };

  return (
    <div className="log-view">
      <div className="log-toolbar">
        <span className="log-title">Logi zdarzeń</span>
        <select value={levelFilter} onChange={e => setLevelFilter(e.target.value)} className="log-filter">
          <option value="SYSTEM">System (domyślny)</option>
          <option value="ALL">Wszystkie + DEBUG</option>
          <option value="BLOCKED">Blokady</option>
          <option value="ERROR">Błędy</option>
        </select>
        <label className="log-autoscroll">
          <input type="checkbox" checked={autoScroll} onChange={e => setAutoScroll(e.target.checked)} />
          Auto-scroll
        </label>
        <button className="log-clear-btn" onClick={onClear}>Wyczyść</button>
      </div>
      <div className="log-entries">
        {filteredLogs.length === 0 ? (
          <div className="log-empty">Brak zdarzeń...</div>
        ) : (
          filteredLogs.map((e, i) => (
            <div key={i} className={`log-entry ${getLevelClass(e.level, e.message)}`}>
              <span className="log-level">{getLevelLabel(e.level)}</span>
              <span className="log-msg">{e.message}</span>
            </div>
          ))
        )}
        <div ref={logEndRef} />
      </div>
    </div>
  );
}
