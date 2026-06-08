import { useState, useEffect, useCallback } from 'react';
import { useT, type Lang } from '../i18n';
import { config } from '../../wailsjs/go/models';
import { GetConfig, SaveConfig, ResetAllowlist } from '../../wailsjs/go/main/App';

interface SettingsViewProps {
  onLanguageChange: (lang: Lang) => void;
}

export function SettingsView({ onLanguageChange }: SettingsViewProps) {
  const { t, lang } = useT();
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
  const [customRulesText, setCustomRulesText] = useState('');
  const [currentLang, setCurrentLang] = useState<Lang>(lang);
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
      setCustomRulesText((c.custom_rules || []).join('\n'));
      const cfgLang = (c as any).language as Lang;
      if (cfgLang === 'pl' || cfgLang === 'en') {
        setCurrentLang(cfgLang);
      }
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
      if (isNaN(wc) || wc < 0) throw new Error(t('settings.error.workers'));
      const cs = parseInt(cacheSize, 10);
      if (isNaN(cs) || cs < 1) throw new Error(t('settings.error.cacheSize'));
      const ttl = parseInt(cacheTtl, 10);
      if (isNaN(ttl) || ttl < 1) throw new Error(t('settings.error.cacheTtl'));
      const interval = parseInt(updateInterval, 10);
      if (isNaN(interval) || interval < 1) throw new Error(t('settings.error.interval'));

      const updated = new config.Config({
        ...cfg,
        allowlist,
        worker_count: wc,
        cache_size: cs,
        cache_ttl: ttl * 60000000000,
        update_interval: interval * 3600000000000,
        log_level: logLevel,
        language: currentLang,
        start_with_system: startWithSystem,
        notifications_enabled: notificationsEnabled,
        minimize_to_tray_on_close: minimizeToTrayOnClose,
        custom_rules: customRulesText
          .split('\n')
          .map(l => l.trim())
          .filter(l => l !== '' && !l.startsWith('#')),
      });
      await SaveConfig(updated);
      setCfg(updated);
      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
    } catch (err: any) {
      setError(err.message || t('settings.error.generic'));
      setTimeout(() => setError(''), 5000);
    }
    setSaving(false);
  };

  if (!cfg) {
    return <div className="sources-loading">{t('settings.loading')}</div>;
  }

  return (
    <div className="settings-view">
      <div className="settings-section">
        <h3>{t('settings.allowlist')}</h3>
        <div className="form-field">
          <span>{t('settings.allowlist.desc')}</span>
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
        <h3>{t('settings.performance')}</h3>
        <div className="settings-row">
          <span className="settings-label">{t('settings.workers')} <code>worker_count</code></span>
          <div>
            <input
              type="number"
              className="settings-input"
              value={workerCount}
              onChange={e => setWorkerCount(e.target.value)}
              min="0"
              max="64"
            />
            <div className="settings-description">{t('settings.workers.desc')}</div>
          </div>
        </div>
        <div className="settings-row">
          <span className="settings-label">{t('settings.cacheSize')} <code>cache_size</code></span>
          <div>
            <input
              type="number"
              className="settings-input"
              value={cacheSize}
              onChange={e => setCacheSize(e.target.value)}
              min="1024"
              max="1048576"
            />
            <div className="settings-description">{t('settings.cacheSize.desc')}</div>
          </div>
        </div>
        <div className="settings-row">
          <span className="settings-label">{t('settings.cacheTtl')} <code>cache_ttl</code></span>
          <div>
            <input
              type="number"
              className="settings-input"
              value={cacheTtl}
              onChange={e => setCacheTtl(e.target.value)}
              min="1"
              max="1440"
            />
            <span style={{ color: 'var(--text-muted)', fontSize: 12, marginLeft: 8 }}>{t('settings.cacheTtl.unit')}</span>
          </div>
        </div>
      </div>

      <div className="settings-section">
        <h3>{t('settings.updates')}</h3>
        <div className="settings-row">
          <span className="settings-label">{t('settings.updateInterval')} <code>update_interval</code></span>
          <div>
            <input
              type="number"
              className="settings-input"
              value={updateInterval}
              onChange={e => setUpdateInterval(e.target.value)}
              min="1"
              max="168"
            />
            <span style={{ color: 'var(--text-muted)', fontSize: 12, marginLeft: 8 }}>{t('settings.updateInterval.unit')}</span>
          </div>
        </div>
      </div>

      <div className="settings-section">
        <h3>{t('settings.logging')}</h3>
        <div className="settings-row">
          <span className="settings-label">{t('settings.logLevel')} <code>log_level</code></span>
          <select
            className="settings-select"
            value={logLevel}
            onChange={e => setLogLevel(e.target.value)}
          >
            <option value="debug">{t('settings.logLevel.debug')}</option>
            <option value="info">{t('settings.logLevel.info')}</option>
            <option value="warn">{t('settings.logLevel.warn')}</option>
            <option value="error">{t('settings.logLevel.error')}</option>
          </select>
        </div>
      </div>

      <div className="settings-section">
        <h3>{t('settings.system')}</h3>
        <div className="settings-row">
          <span className="settings-label">{t('settings.system')}</span>
          <label className="settings-checkbox-label source-toggle">
            <input
              type="checkbox"
              checked={startWithSystem}
              onChange={e => setStartWithSystem(e.target.checked)}
            />
            <span className="toggle-track">
              <span className="toggle-indicator" />
            </span>
            <span>{t('settings.startWithSystem')}</span>
          </label>
        </div>
      </div>

      <div className="settings-section">
        <h3>{t('settings.notifications')}</h3>
        <div className="settings-row">
          <span className="settings-label">{t('settings.notifications')}</span>
          <label className="settings-checkbox-label source-toggle">
            <input
              type="checkbox"
              checked={notificationsEnabled}
              onChange={e => setNotificationsEnabled(e.target.checked)}
            />
            <span className="toggle-track">
              <span className="toggle-indicator" />
            </span>
            <span>{t('settings.notifications.desc')}</span>
          </label>
        </div>
      </div>

      <div className="settings-section">
        <h3>{t('settings.close')}</h3>
        <div className="settings-row">
          <span className="settings-label">{t('settings.close.label')}</span>
          <label className="settings-checkbox-label source-toggle">
            <input
              type="checkbox"
              checked={minimizeToTrayOnClose}
              onChange={e => setMinimizeToTrayOnClose(e.target.checked)}
            />
            <span className="toggle-track">
              <span className="toggle-indicator" />
            </span>
            <span>{t('settings.close.desc')}</span>
          </label>
        </div>
      </div>

      <div className="settings-section">
        <h3>{t('settings.customRules')}</h3>
        <div className="form-field">
          <span>{t('settings.customRules.desc')}</span>
          <textarea
            className="settings-textarea"
            value={customRulesText}
            onChange={e => setCustomRulesText(e.target.value)}
            placeholder="10.0.0.0/8&#10;185.220.101.0/24&#10;5.188.62.0-5.188.62.255&#10;# komentarz"
            rows={6}
          />
          <div className="settings-description">{t('settings.customRules.hint')}</div>
        </div>
      </div>

      <div className="settings-section">
        <h3>{t('settings.language')}</h3>
        <div className="settings-row">
          <span className="settings-label">{t('settings.language')}</span>
          <select
            className="settings-select"
            value={currentLang}
            onChange={e => {
              const newLang = e.target.value as Lang;
              setCurrentLang(newLang);
              onLanguageChange(newLang);
            }}
          >
            <option value="pl">{t('settings.language.pl')}</option>
            <option value="en">{t('settings.language.en')}</option>
          </select>
        </div>
      </div>

      <div className="settings-actions">
        <button className="btn-primary" onClick={handleSave} disabled={saving}>
          {saving ? t('settings.saving') : t('settings.save')}
        </button>
        <button className="btn-secondary reset-btn" onClick={async () => {
          if (!confirm(t('settings.resetAllowlist.confirm'))) return;
          try {
            await ResetAllowlist();
            await loadConfig();
            setSaved(true);
            setTimeout(() => setSaved(false), 3000);
          } catch (e) {
            setError(t('settings.resetAllowlist.error'));
            setTimeout(() => setError(''), 5000);
          }
        }}>
          {t('settings.resetAllowlist')}
        </button>
        {saved && <span className="settings-saved">{t('settings.saved')}</span>}
        {error && <span className="settings-error">{error}</span>}
      </div>
    </div>
  );
}
