import { useState } from 'react';
import { useT } from '../i18n';

export interface AddSourceForm {
  name: string;
  url: string;
  format: number;
  apiKey: string;
  description: string;
}

interface AddSourceDialogProps {
  onClose: () => void;
  onSave: (src: AddSourceForm) => void;
}

export function AddSourceDialog({ onClose, onSave }: AddSourceDialogProps) {
  const { t } = useT();
  const FORMAT_OPTIONS = [
    { value: 3, label: t('addSource.format.cidr') },
    { value: 4, label: t('addSource.format.range') },
    { value: 1, label: t('addSource.format.p2p') },
    { value: 2, label: t('addSource.format.dat') },
  ];
  const [form, setForm] = useState<AddSourceForm>({
    name: '', url: '', format: 3, apiKey: '', description: '',
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.name.trim() || !form.url.trim()) return;
    onSave(form);
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <h3>{t('addSource.title')}</h3>
          <button className="modal-close" onClick={onClose}>&times;</button>
        </div>
        <form onSubmit={handleSubmit}>
          <div className="modal-body">
            <label className="form-field">
              <span>{t('addSource.name')}</span>
              <input
                type="text" value={form.name}
                onChange={e => setForm({ ...form, name: e.target.value })}
                placeholder={t('addSource.name.placeholder')}
                required
              />
            </label>
            <label className="form-field">
              <span>{t('addSource.url')}</span>
              <input
                type="url" value={form.url}
                onChange={e => setForm({ ...form, url: e.target.value })}
                placeholder={t('addSource.url.placeholder')}
                required
              />
            </label>
            <label className="form-field">
              <span>{t('addSource.format')}</span>
              <select value={form.format} onChange={e => setForm({ ...form, format: Number(e.target.value) })}>
                {FORMAT_OPTIONS.map(o => (
                  <option key={o.value} value={o.value}>{o.label}</option>
                ))}
              </select>
            </label>
            <label className="form-field">
              <span>{t('addSource.apiKey')}</span>
              <input
                type="text" value={form.apiKey}
                onChange={e => setForm({ ...form, apiKey: e.target.value })}
                placeholder={t('addSource.apiKey.placeholder')}
              />
            </label>
            <label className="form-field">
              <span>{t('addSource.description')}</span>
              <textarea
                value={form.description}
                onChange={e => setForm({ ...form, description: e.target.value })}
                placeholder={t('addSource.description.placeholder')}
                rows={2}
              />
            </label>
          </div>
          <div className="modal-footer">
            <button type="button" className="btn-secondary" onClick={onClose}>{t('addSource.cancel')}</button>
            <button type="submit" className="btn-primary">{t('addSource.submit')}</button>
          </div>
        </form>
      </div>
    </div>
  );
}
