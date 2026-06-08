import { useState, useEffect, useCallback, useRef } from 'react';
import './App.css';
import appIcon from './assets/ikona.png';
import { GetStats, GetLogs, GetConfig, IsProtectionEnabled, ToggleProtection, TriggerUpdate, GetDatabaseInfo, GetCacheInfo, MinimizeToTray, QuitApp } from "../wailsjs/go/main/App";
import { EventsOn, InitializeNotifications, SendNotification, CleanupNotifications } from '../wailsjs/runtime/runtime';
import { filter, logger } from "../wailsjs/go/models";
import { Dashboard } from './components/Dashboard';
import { SourcesView } from './components/SourcesView';
import { SettingsView } from './components/SettingsView';
import { LogView } from './components/LogView';
import { ChartsView, type Sample, type BlockedEntry } from './components/ChartsView';
import { I18nProvider, useT, type Lang } from './i18n';

type Stats = filter.Stats;
type LogEntry = logger.LogEntry;
type Tab = 'dashboard' | 'sources' | 'settings' | 'charts';

// ─── Main App ────────────────────────────────────────────

function AppInner() {
  const { t, setLang, lang } = useT();
  const [stats, setStats] = useState<Stats | null>(null);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [protected_, setProtected_] = useState(false);
  const [dbInfo, setDbInfo] = useState<Record<string, any>>({});
  const [uptime, setUptime] = useState('0s');
  const [tab, setTab] = useState<Tab>('dashboard');
  const [cacheInfo, setCacheInfo] = useState<Record<string, any>>({});
  const [history, setHistory] = useState<Sample[]>([]);
  const [blockedEntries, setBlockedEntries] = useState<BlockedEntry[]>([]);
  const blockIdRef = useRef(0);
  const prevStatsRef = useRef<Stats | null>(null);
  const prevTimeRef = useRef<number>(0);
  const collectingRef = useRef(false);
  const [rangeDiffs, setRangeDiffs] = useState<Record<string, number>>({});
  const [closeDialog, setCloseDialog] = useState(false);
  // Pomija pierwszy event update-status (startowy), żeby nie wysyłać toasta przy starcie
  const startupRef = useRef(true);

  const refresh = useCallback(async () => {
    try {
      const s = await GetStats();
      setStats(s);
      const p = await IsProtectionEnabled();
      setProtected_(p);
      const d = await GetDatabaseInfo();
      setDbInfo(d);
      const ci = await GetCacheInfo();
      setCacheInfo(ci);
      const l = await GetLogs(200);
      setLogs(l.reverse()); // newest first, matching event prepend
    } catch (err) {
      console.error('refresh error', err);
    }
  }, []);

  useEffect(() => {
    // Initial data fetch
    refresh();

    // Initialize Windows toast notifications
    InitializeNotifications().catch(() => {});

    // Live event listeners — replaces 2s polling (A7)
    const cancelStats = EventsOn("stats", (s: Stats) => {
      setStats(s);

      // Calculate PPS delta for chart history
      if (collectingRef.current && prevStatsRef.current) {
        const now = Date.now();
        const elapsed = (now - prevTimeRef.current) / 1000;
        if (elapsed >= 0.5) {
          const blockedDelta = s.blocked - prevStatsRef.current.blocked;
          const allowedDelta = s.allowed - prevStatsRef.current.allowed;
          if (blockedDelta >= 0 && allowedDelta >= 0) {
            setHistory(prev => {
              const next = [...prev, {
                time: now,
                blockedPPS: blockedDelta / elapsed,
                allowedPPS: allowedDelta / elapsed,
              }];
              const cutoff = now - 30 * 60 * 1000;
              return next.filter(h => h.time >= cutoff);
            });
          }
        }
      }
      prevStatsRef.current = s;
      prevTimeRef.current = Date.now();
    });

    const cancelLog = EventsOn("log", (entry: LogEntry) => {
      setLogs(prev => {
        const next = [entry, ...prev];
        return next.length > 200 ? next.slice(0, 200) : next;
      });
      // Split BLOCK messages into separate list for ChartsView
      if (entry.message.startsWith('BLOCK')) {
        const m = entry.message.match(/^BLOCK (\S+) → (\S+) \[(\w+)\]$/);
        if (m) {
          const id = ++blockIdRef.current;
          setBlockedEntries(prev => {
            const next = [{ id, timestamp: Date.now(), srcIP: m[1], dstIP: m[2], proto: m[3] }, ...prev];
            return next.length > 500 ? next.slice(0, 500) : next;
          });
        }
      }
    });

    const cancelProtection = EventsOn("protection", (enabled: boolean) => {
      setProtected_(enabled);
    });

    const cancelDbInfo = EventsOn("db-info", (info: Record<string, any>) => {
      setDbInfo(info);
    });

    const cancelCacheInfo = EventsOn("cache-info", (info: Record<string, any>) => {
      setCacheInfo(info);
    });

    const cancelCloseRequest = EventsOn("close-request", () => {
      setCloseDialog(true);
    });

    const cancelUpdateStatus = EventsOn("update-status", async (data: any) => {
      setUpdating(false);
      // Store range diffs for SourcesView badges
      if (data?.diffs) {
        setRangeDiffs(data.diffs);
      }
      // Skip the first event (startup update) — nie wysyłamy toasta przy starcie
      if (startupRef.current) {
        startupRef.current = false;
        return;
      }
      if (!data?.ranges) return;
      try {
        const cfg = await GetConfig();
        if (cfg.notifications_enabled !== false) {
          await SendNotification({
            id: 'update-complete',
            title: 'GO PeerBlock',
            body: t('notification.body', { ranges: data.ranges.toLocaleString() }),
          });
        }
      } catch {}
    });

    return () => {
      cancelStats();
      cancelLog();
      cancelProtection();
      cancelDbInfo();
      cancelCacheInfo();
      cancelUpdateStatus();
      cancelCloseRequest();
      CleanupNotifications().catch(() => {});
    };
  }, [refresh]);

  useEffect(() => {
    if (!stats?.started_at || stats.started_at === 0) return;
    const start = Math.floor(stats.started_at / 1_000_000);
    const update = () => {
      const diff = Math.floor((Date.now() - start) / 1000);
      const h = Math.floor(diff / 3600);
      const m = Math.floor((diff % 3600) / 60);
      const s = diff % 60;
      setUptime(h > 0 ? `${h}h ${m}m` : m > 0 ? `${m}m ${s}s` : `${s}s`);
    };
    update();
    const iv = setInterval(update, 1000);
    return () => clearInterval(iv);
  }, [stats?.started_at]);

  // Pause chart data collection when the tab isn't active
  useEffect(() => {
    collectingRef.current = tab === 'charts';
  }, [tab]);

  const handleToggle = async () => {
    await ToggleProtection();
    setProtected_(!protected_);
  };

  const [updating, setUpdating] = useState(false);

  const handleUpdate = async () => {
    setUpdating(true);
    await TriggerUpdate();
    // Safety timeout: odblokuj przycisk po 60s nawet jeśli event nie przyjdzie (np. Go padnie)
    setTimeout(() => setUpdating(false), 60_000);
  };

  const handleClearLogs = () => {
    setLogs([]);
    setBlockedEntries([]);
  };

  const [appLang, setAppLang] = useState<Lang>('en');

  // Load language from backend config on startup
  useEffect(() => {
    (async () => {
      try {
        const cfg = await GetConfig();
        const savedLang = (cfg as any).language as Lang;
        if (savedLang === 'pl' || savedLang === 'en') {
          setAppLang(savedLang);
          setLang(savedLang);
        }
      } catch {}
    })();
  }, []);

  return (
    <div className="app">
      <header className="app-header">
        <div className="header-left">
          <img className="app-logo" src={appIcon} alt="GO PeerBlock" />
          <span className="app-subtitle">{t('app.subtitle')}</span>
        </div>
        <div className="header-right">
          <button className="update-btn" onClick={handleUpdate} title={t('app.update.title')} disabled={updating}>
            {updating ? '⏳' : '↻'} {t('app.update')}
          </button>
          <button className="tray-btn" onClick={MinimizeToTray} title={t('app.tray.title')}>
            ⬇
          </button>
        </div>
      </header>

      <nav className="tab-nav">
        <button
          className={`tab-btn ${tab === 'dashboard' ? 'active' : ''}`}
          onClick={() => setTab('dashboard')}
        >{t('tab.dashboard')}</button>
        <button
          className={`tab-btn ${tab === 'sources' ? 'active' : ''}`}
          onClick={() => setTab('sources')}
        >{t('tab.sources')}</button>
        <button
          className={`tab-btn ${tab === 'charts' ? 'active' : ''}`}
          onClick={() => setTab('charts')}
        >{t('tab.charts')}</button>
        <button
          className={`tab-btn ${tab === 'settings' ? 'active' : ''}`}
          onClick={() => setTab('settings')}
        >{t('tab.settings')}</button>
      </nav>

      <main className="app-main">
        {tab === 'dashboard' && (
          <Dashboard
            stats={stats} uptime={uptime} dbInfo={dbInfo} cacheInfo={cacheInfo}
            protected_={protected_} onToggle={handleToggle}
          />
        )}
        {tab === 'sources' && <SourcesView onUpdate={handleUpdate} updating={updating} rangeDiffs={rangeDiffs} />}
        {tab === 'settings' && <SettingsView onLanguageChange={setLang} />}
        {tab === 'charts' && <ChartsView history={history} blockedEntries={blockedEntries} />}
        <LogView logs={logs} onClear={handleClearLogs} />
      </main>

      {/* Close dialog */}
      {closeDialog && (
        <div className="modal-overlay" onClick={() => setCloseDialog(false)}>
          <div className="modal modal-close" onClick={e => e.stopPropagation()}>
            <div className="modal-body">
              <div className="close-dialog-icon">❓</div>
              <p className="close-dialog-title">{t('closeDialog.title')}</p>
              <p className="close-dialog-desc">
                {t('closeDialog.desc')}
              </p>
              <div className="close-dialog-actions">
                <button className="close-dialog-btn tray" onClick={() => { setCloseDialog(false); MinimizeToTray(); }}>
                  {t('closeDialog.tray')}
                </button>
                <button className="close-dialog-btn quit" onClick={() => { setCloseDialog(false); QuitApp(); }}>
                  {t('closeDialog.quit')}
                </button>
                <button className="close-dialog-btn cancel" onClick={() => setCloseDialog(false)}>
                  {t('closeDialog.cancel')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      <footer className="status-bar">
        <span className={`status-dot ${protected_ ? 'active' : 'inactive'}`} />
        <span>{protected_ ? t('protection.active') : t('protection.inactive')}</span>
        {stats && (() => {
          const total = stats.blocked + stats.allowed;
          const elapsed = stats.started_at > 0
            ? (Date.now() - Math.floor(stats.started_at / 1_000_000)) / 1000
            : 0;
          const pps = elapsed > 0 ? (total / elapsed).toFixed(1) : '0.0';
          return (
            <span className="status-fps">
              {t('status.packets', { total: total.toLocaleString(), pps })}
            </span>
          );
        })()}
        <span className="status-tab">{
          tab === 'dashboard' ? t('status.tab.dashboard') : tab === 'sources' ? t('status.tab.sources') : tab === 'charts' ? t('status.tab.charts') : t('status.tab.settings')
        }</span>
      </footer>
    </div>
  );
}

function App() {
  return (
    <I18nProvider initialLang="en">
      <AppInner />
    </I18nProvider>
  );
}

export default App;
