import { useState, useEffect, useCallback } from 'react';
import { config } from '../../wailsjs/go/models';
import { GetConfig, SaveConfig, ResetAllowlist } from '../../wailsjs/go/main/App';

export function SettingsView() {
  const [cfg, setCfg] = useState<config.Config | null>(null);
  const [allowlistText, setAllowlistText] = useState('');
  const [workerCount, setWorkerCount] = useState('0');
  const [cacheSize, setCacheSize] = useState('65536');
  const [cacheTtl, setCacheTtl] = useState('5');
  const [updateInterval, setUpdateInterval] = useState('24');
  const [logLevel, setLogLevel] = useState('info');
  const [startWithSystem, setStartWithSystem] = useState(false);
  const [notificationsEnabled, setNotificationsEnabled] = useState(true);
  const [minimizeToTrayOnClose, setMinimizeToTrayOnClose] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState('');

  const loadConfig = useCallback(async () => {
    try {
      const c = await GetConfig();
      setCfg(c);
      setAllowlistText((c.allowlist || []).join('\n'));
      setWorkerCount(String(c.worker_count ?? 0));
      setCacheSize(String(c.cache_size ?? 65536));
      const ttlNs = c.cache_ttl ?? 300000000000;
      setCacheTtl(String(Math.round(ttlNs / 60000000000)));
      const intervalNs = c.update_interval ?? 86400000000000;
      setUpdateInterval(String(Math.round(intervalNs / 3600000000000)));
      setLogLevel(c.log_level || 'info');
      setStartWithSystem(!!c.start_with_system);
      setNotificationsEnabled(c.notifications_enabled !== false);
      setMinimizeToTrayOnClose(!!c.minimize_to_tray_on_close);
    } catch (err) {
      console.error('load config error', err);
    }
  }, []);

  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  const handleSave = async () => {
    if (!cfg) return;
    setSaving(true);
    setSaved(false);
    setError('');
    try {
      const allowlist = allowlistText
        .split('\n')
        .map(l => l.trim())
        .filter(l => l !== '' && !l.startsWith('#'));
      
      const wc = parseInt(workerCount, 10);
      if (isNaN(wc) || wc < 0) throw new Error('Nieprawidłowa liczba workerów');
      const cs = parseInt(cacheSize, 10);
      if (isNaN(cs) || cs < 1) throw new Error('Nieprawidłowy rozmiar cache');
      const ttl = parseInt(cacheTtl, 10);
      if (isNaN(ttl) || ttl < 1) throw new Error('Nieprawidłowy TTL');
      const interval = parseInt(updateInterval, 10);
      if (isNaN(interval) || interval < 1) throw new Error('Nieprawidłowy interwał');

      const updated = new config.Config({
        ...cfg,
        allowlist,
        worker_count: wc,
        cache_size: cs,
        cache_ttl: ttl * 60000000000,
        update_interval: interval * 3600000000000,
        log_level: logLevel,
        start_with_system: startWithSystem,
        notifications_enabled: notificationsEnabled,
        minimize_to_tray_on_close: minimizeToTrayOnClose,
      });
      await SaveConfig(updated);
      setCfg(updated);
      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
    } catch (err: any) {
      setError(err.message || 'Błąd zapisu');
      setTimeout(() => setError(''), 5000);
    }
    setSaving(false);
  };

  if (!cfg) {
    return <div className="sources-loading">Ładowanie ustawień...</div>;
  }

  return (
    <div className="settings-view">
      <div className="settings-section">
        <h3>Allowlista</h3>
        <div className="form-field">
          <span>Adresy IP, CIDR lub domeny (jeden na linię)</span>
          <textarea
            className="settings-textarea"
            value={allowlistText}
            onChange={e => setAllowlistText(e.target.value)}
            placeholder="8.8.8.8&#10;192.168.0.0/16&#10;*.example.com"
            rows={6}
          />
        </div>
      </div>

      <div className="settings-section">
        <h3>Wydajność</h3>
        <div className="settings-row">
          <span className="settings-label">Liczba workerów <code>worker_count</code></span>
          <div>
            <input
              type="number"
              className="settings-input"
              value={workerCount}
              onChange={e => setWorkerCount(e.target.value)}
              min="0"
              max="64"
            />
            <div className="settings-description">0 = automatycznie (liczba CPU)</div>
          </div>
        </div>
        <div className="settings-row">
          <span className="settings-label">Rozmiar cache <code>cache_size</code></span>
          <div>
            <input
              type="number"
              className="settings-input"
              value={cacheSize}
              onChange={e => setCacheSize(e.target.value)}
              min="1024"
              max="1048576"
            />
            <div className="settings-description">Liczba wpisów w cache decyzji</div>
          </div>
        </div>
        <div className="settings-row">
          <span className="settings-label">Cache TTL <code>cache_ttl</code></span>
          <div>
            <input
              type="number"
              className="settings-input"
              value={cacheTtl}
              onChange={e => setCacheTtl(e.target.value)}
              min="1"
              max="1440"
            />
            <span style={{ color: 'var(--text-muted)', fontSize: 12, marginLeft: 8 }}>minut</span>
          </div>
        </div>
      </div>

      <div className="settings-section">
        <h3>Aktualizacje</h3>
        <div className="settings-row">
          <span className="settings-label">Interwał aktualizacji <code>update_interval</code></span>
          <div>
            <input
              type="number"
              className="settings-input"
              value={updateInterval}
              onChange={e => setUpdateInterval(e.target.value)}
              min="1"
              max="168"
            />
            <span style={{ color: 'var(--text-muted)', fontSize: 12, marginLeft: 8 }}>godzin</span>
          </div>
        </div>
      </div>

      <div className="settings-section">
        <h3>Logowanie</h3>
        <div className="settings-row">
          <span className="settings-label">Poziom logowania <code>log_level</code></span>
          <select
            className="settings-select"
            value={logLevel}
            onChange={e => setLogLevel(e.target.value)}
          >
            <option value="debug">DEBUG — wszystko</option>
            <option value="info">INFO — informacje + błędy</option>
            <option value="warn">WARN — tylko ostrzeżenia + błędy</option>
            <option value="error">ERROR — tylko błędy</option>
          </select>
        </div>
      </div>

      <div className="settings-section">
        <h3>System</h3>
        <div className="settings-row">
          <span className="settings-label">Uruchamianie</span>
          <label className="settings-checkbox-label source-toggle">
            <input
              type="checkbox"
              checked={startWithSystem}
              onChange={e => setStartWithSystem(e.target.checked)}
            />
            <span className="toggle-track">
              <span className="toggle-indicator" />
            </span>
            <span>Uruchamiaj z systemem Windows</span>
          </label>
        </div>
      </div>

      <div className="settings-section">
        <h3>Powiadomienia</h3>
        <div className="settings-row">
          <span className="settings-label">Aktualizacje list</span>
          <label className="settings-checkbox-label source-toggle">
            <input
              type="checkbox"
              checked={notificationsEnabled}
              onChange={e => setNotificationsEnabled(e.target.checked)}
            />
            <span className="toggle-track">
              <span className="toggle-indicator" />
            </span>
            <span>Powiadom o zakończeniu aktualizacji list IP</span>
          </label>
        </div>
      </div>

      <div className="settings-section">
        <h3>Zamykanie aplikacji</h3>
        <div className="settings-row">
          <span className="settings-label">Przycisk X</span>
          <label className="settings-checkbox-label source-toggle">
            <input
              type="checkbox"
              checked={minimizeToTrayOnClose}
              onChange={e => setMinimizeToTrayOnClose(e.target.checked)}
            />
            <span className="toggle-track">
              <span className="toggle-indicator" />
            </span>
            <span>Nie pytaj — minimalizuj do tray</span>
          </label>
        </div>
      </div>

      <div className="settings-actions">
        <button className="btn-primary" onClick={handleSave} disabled={saving}>
          {saving ? '⏳ Zapisywanie...' : '💾 Zapisz ustawienia'}
        </button>
        <button className="btn-secondary reset-btn" onClick={async () => {
          if (!confirm('Czy na pewno chcesz przywrócić domyślną allowlistę? Twoje własne wpisy zostaną usunięte.')) return;
          try {
            await ResetAllowlist();
            await loadConfig();
            setSaved(true);
            setTimeout(() => setSaved(false), 3000);
          } catch (e) {
            setError('Reset allowlisty nie powiódł się');
            setTimeout(() => setError(''), 5000);
          }
        }}>
          Przywróć domyślną allowlistę
        </button>
        {saved && <span className="settings-saved">✅ Zapisano!</span>}
        {error && <span className="settings-error">{error}</span>}
      </div>
    </div>
  );
}
