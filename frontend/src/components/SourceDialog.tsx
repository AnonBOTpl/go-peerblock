interface SourceDialogProps {
  ip: string;
  sources: string[];
  onClose: () => void;
}

export function SourceDialog({ ip, sources, onClose }: SourceDialogProps) {
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal modal-source" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <h3>Źródła blokady</h3>
          <button className="modal-close-btn" onClick={onClose}>✕</button>
        </div>
        <div className="modal-body">
          <p className="source-dialog-ip">
            <span className="source-dialog-label">IP:</span>
            <span className="source-dialog-value">{ip}</span>
          </p>
          <div className="source-dialog-list">
            <p className="source-dialog-list-label">Blokowany przez:</p>
            {sources.length === 0 ? (
              <p className="source-dialog-empty">Brak dopasowanych źródeł</p>
            ) : (
              sources.map((name, i) => (
                <div key={i} className="source-dialog-item">
                  <span className="source-dialog-bullet">•</span>
                  <span>{name}</span>
                </div>
              ))
            )}
          </div>
        </div>
        <div className="modal-footer">
          <button className="btn-secondary" onClick={onClose}>Zamknij</button>
        </div>
      </div>
    </div>
  );
}
