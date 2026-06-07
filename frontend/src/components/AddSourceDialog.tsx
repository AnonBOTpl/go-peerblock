import { useState } from 'react';

export interface AddSourceForm {
  name: string;
  url: string;
  format: number;
  apiKey: string;
  description: string;
}

const FORMAT_OPTIONS = [
  { value: 3, label: 'CIDR (1.2.3.0/24)' },
  { value: 4, label: 'Zakres (1.2.3.0-1.2.3.255)' },
  { value: 1, label: 'P2P Text (Level1:1.2.3.0-1.2.3.255)' },
  { value: 2, label: 'DAT (1.2.3.0 - 1.2.3.255 , 100 , Name)' },
];

interface AddSourceDialogProps {
  onClose: () => void;
  onSave: (src: AddSourceForm) => void;
}

export function AddSourceDialog({ onClose, onSave }: AddSourceDialogProps) {
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
          <h3>Dodaj źródło</h3>
          <button className="modal-close" onClick={onClose}>&times;</button>
        </div>
        <form onSubmit={handleSubmit}>
          <div className="modal-body">
            <label className="form-field">
              <span>Nazwa</span>
              <input
                type="text" value={form.name}
                onChange={e => setForm({ ...form, name: e.target.value })}
                placeholder="np. moja-lista"
                required
              />
            </label>
            <label className="form-field">
              <span>URL</span>
              <input
                type="url" value={form.url}
                onChange={e => setForm({ ...form, url: e.target.value })}
                placeholder="https://example.com/blocklist.txt"
                required
              />
            </label>
            <label className="form-field">
              <span>Format</span>
              <select value={form.format} onChange={e => setForm({ ...form, format: Number(e.target.value) })}>
                {FORMAT_OPTIONS.map(o => (
                  <option key={o.value} value={o.value}>{o.label}</option>
                ))}
              </select>
            </label>
            <label className="form-field">
              <span>API Key (opcjonalne)</span>
              <input
                type="text" value={form.apiKey}
                onChange={e => setForm({ ...form, apiKey: e.target.value })}
                placeholder="Zostaw puste jeśli nie wymagane"
              />
            </label>
            <label className="form-field">
              <span>Opis (opcjonalne)</span>
              <textarea
                value={form.description}
                onChange={e => setForm({ ...form, description: e.target.value })}
                placeholder="Co ta lista blokuje?"
                rows={2}
              />
            </label>
          </div>
          <div className="modal-footer">
            <button type="button" className="btn-secondary" onClick={onClose}>Anuluj</button>
            <button type="submit" className="btn-primary">Dodaj źródło</button>
          </div>
        </form>
      </div>
    </div>
  );
}
