import { useState, useEffect, useCallback } from 'react';
import { config, updater } from '../../wailsjs/go/models';
import { GetConfig, SaveConfig } from '../../wailsjs/go/main/App';
import { AddSourceDialog, AddSourceForm } from './AddSourceDialog';

type Source = updater.Source;

const FORMAT_LABELS: Record<number, string> = {
  1: 'P2P Text',
  2: 'DAT',
  3: 'CIDR',
  4: 'Zakres IP',
};

interface SourcesViewProps {
  onUpdate: () => void;
  updating: boolean;
  rangeDiffs: Record<string, number>;
}

export function SourcesView({ onUpdate, updating, rangeDiffs }: SourcesViewProps) {
  const [cfg, setCfg] = useState<config.Config | null>(null);
  const [saving, setSaving] = useState(false);
  const [showAdd, setShowAdd] = useState(false);
  const [deletingIndex, setDeletingIndex] = useState<number | null>(null);

  const loadConfig = useCallback(async () => {
    try {
      const c = await GetConfig();
      setCfg(c);
    } catch (err) {
      console.error('load config error', err);
    }
  }, []);

  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  const saveSources = async (sources: Source[]) => {
    const updated = new config.Config({ ...cfg, sources });
    setCfg(updated);
    setSaving(true);
    try {
      await SaveConfig(updated);
    } catch (err) {
      console.error('save config error', err);
    }
    setSaving(false);
  };

  const toggleSource = async (index: number) => {
    if (!cfg) return;
    const sources = cfg.sources.map((s, i) =>
      i === index ? new updater.Source({ ...s, enabled: !s.enabled }) : s
    );
    await saveSources(sources);
  };

  const handleAdd = async (form: AddSourceForm) => {
    if (!cfg) return;
    const sources = [...cfg.sources, new updater.Source({
      name: form.name,
      url: form.url,
      format: form.format,
      enabled: true,
      api_key: form.apiKey || '',
      description: form.description || '',
    })];
    await saveSources(sources);
    setShowAdd(false);
  };

  const handleDelete = async (index: number) => {
    if (!cfg) return;
    const sources = cfg.sources.filter((_, i) => i !== index);
    await saveSources(sources);
    setDeletingIndex(null);
  };

  if (!cfg) {
    return <div className="sources-loading">Ładowanie konfiguracji...</div>;
  }

  return (
    <div className="sources-view">
      <div className="sources-header">
        <h2>Źródła list IP ({cfg.sources.length})</h2>
        <div className="sources-actions">
          <button className="btn-secondary" onClick={() => setShowAdd(true)}>
            + Dodaj źródło
          </button>
          <button className="update-btn" onClick={onUpdate} disabled={updating}>
            {updating ? '⏳' : '↻'} Aktualizuj
          </button>
        </div>
      </div>
      <p className="sources-desc">
        Włącz lub wyłącz źródła blokad IP. Możesz dodać własne źródła z opcjonalnym kluczem API.
        Kliknij "Aktualizuj teraz" aby pobrać wybrane listy.
      </p>
      <div className="sources-list">
        {cfg.sources.map((src, i) => (
          <div key={i} className={`source-card ${src.enabled ? 'enabled' : 'disabled'}`}>
            <div className="source-main">
              <div className="source-info">
                <div className="source-name">{src.name}</div>
                {src.description && (
                  <div className="source-desc">{src.description}</div>
                )}
                <div className="source-url" title={src.url}>{src.url}</div>
                <div className="source-meta">
                  <span className="source-format-badge">{FORMAT_LABELS[src.format] || 'CIDR'}</span>
                  {src.api_key && (
                    <span className="source-api-badge" title={`API Key: ${src.api_key.substring(0, 8)}...`}>
                      🔑 API
                    </span>
                  )}
                  {src.range_count > 0 && (
                    <span className="source-range-count">
                      {src.range_count.toLocaleString()} zakresów
                      {rangeDiffs[src.name] !== undefined && (
                        <span className={`range-diff ${rangeDiffs[src.name] > 0 ? 'up' : rangeDiffs[src.name] < 0 ? 'down' : 'same'}`}>
                          {' '}{rangeDiffs[src.name] > 0 ? '▲' : rangeDiffs[src.name] < 0 ? '▼' : '—'} {Math.abs(rangeDiffs[src.name]).toLocaleString()}
                        </span>
                      )}
                    </span>
                  )}
                  {src.last_sync && (
                    <span className="source-last-sync">
                      Ostatnia: {new Date(src.last_sync).toLocaleString('pl-PL')}
                    </span>
                  )}
                </div>
              </div>
              <div className="source-controls">
                <label className="source-toggle" onClick={e => e.stopPropagation()}>
                  <input
                    type="checkbox"
                    checked={src.enabled}
                    onChange={() => toggleSource(i)}
                  />
                  <span className="toggle-track">
                    <span className="toggle-indicator" />
                  </span>
                  <span className="toggle-status">{src.enabled ? 'Aktywne' : 'Wył.'}</span>
                </label>
                <button
                  className="source-delete-btn"
                  onClick={() => setDeletingIndex(i)}
                  title="Usuń źródło"
                >
                  🗑
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
      {saving && <div className="sources-saving">Zapisywanie...</div>}

      {/* Delete confirmation */}
      {deletingIndex !== null && (
        <div className="modal-overlay" onClick={() => setDeletingIndex(null)}>
          <div className="modal modal-sm" onClick={e => e.stopPropagation()}>
            <p className="modal-confirm-text">
              Usunąć źródło <strong>{cfg.sources[deletingIndex]?.name}</strong>?
            </p>
            <div className="modal-footer">
              <button className="btn-secondary" onClick={() => setDeletingIndex(null)}>Anuluj</button>
              <button className="btn-danger" onClick={() => handleDelete(deletingIndex)}>Usuń</button>
            </div>
          </div>
        </div>
      )}

      {/* Add source dialog */}
      {showAdd && (
        <AddSourceDialog
          onClose={() => setShowAdd(false)}
          onSave={handleAdd}
        />
      )}
    </div>
  );
}
