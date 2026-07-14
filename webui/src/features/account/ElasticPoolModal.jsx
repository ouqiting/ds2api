import { X } from 'lucide-react'
import clsx from 'clsx'
import { useState } from 'react'

export default function ElasticPoolModal({
    show,
    t,
    elasticPool,
    setElasticPool,
    loading,
    onClose,
    onSave,
}) {
    if (!show) {
        return null
    }

    const [error, setError] = useState(null)
    const perPool = elasticPool.per_pool ? 'per_pool' : 'global'
    const enabledCount = perPool === 'global' ? (Number(elasticPool.global_count) || 0) : 'X'

    const handleSave = async () => {
        setError(null)
        const err = await onSave()
        if (err) {
            setError(err)
        }
    }

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4 animate-in fade-in">
            <div className="bg-card w-full max-w-md rounded-xl border border-border shadow-2xl overflow-hidden animate-in zoom-in-95">
                <div className="p-4 border-b border-border flex justify-between items-center">
                    <h3 className="font-semibold">{t('accountManager.elasticPoolSettings')}</h3>
                    <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
                        <X className="w-5 h-5" />
                    </button>
                </div>
                <div className="p-6 space-y-4">
                    <div className="flex items-center justify-between">
                        <label className="text-sm font-medium">{t('accountManager.elasticPoolEnable')}</label>
                        <button
                            type="button"
                            role="switch"
                            aria-checked={elasticPool.enabled}
                            onClick={() => setElasticPool({ ...elasticPool, enabled: !elasticPool.enabled })}
                            className={clsx(
                                "relative inline-flex h-5 w-9 items-center rounded-full transition-colors shrink-0",
                                elasticPool.enabled ? "bg-primary" : "bg-muted-foreground/30"
                            )}
                        >
                            <span className={clsx(
                                "inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform shadow-sm",
                                elasticPool.enabled ? "translate-x-[18px]" : "translate-x-1"
                            )} />
                        </button>
                    </div>

                    <div>
                        <label className="block text-sm font-medium mb-1.5">{t('accountManager.elasticPoolMode')}</label>
                        <select
                            className="input-field"
                            value={perPool}
                            onChange={e => setElasticPool({ ...elasticPool, per_pool: e.target.value === 'per_pool' })}
                        >
                            <option value="global">{t('accountManager.elasticPoolModeGlobal')}</option>
                            <option value="per_pool">{t('accountManager.elasticPoolModePerPool')}</option>
                        </select>
                    </div>

                    {perPool === 'global' ? (
                        <div>
                            <label className="block text-sm font-medium mb-1.5">{t('accountManager.elasticPoolGlobalCount')}</label>
                            <input
                                type="number"
                                min="0"
                                className="input-field"
                                placeholder={t('accountManager.elasticPoolGlobalCountPlaceholder')}
                                value={elasticPool.global_count}
                                onChange={e => setElasticPool({ ...elasticPool, global_count: e.target.value })}
                            />
                        </div>
                    ) : (
                        <>
                            <div>
                                <label className="block text-sm font-medium mb-1.5">{t('accountManager.elasticPoolDefaultCount')}</label>
                                <input
                                type="number"
                                min="0"
                                className="input-field"
                                value={elasticPool.default_count}
                                onChange={e => setElasticPool({ ...elasticPool, default_count: e.target.value })}
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium mb-1.5">{t('accountManager.elasticPoolNoToolsCount')}</label>
                                <input
                                type="number"
                                min="0"
                                className="input-field"
                                value={elasticPool.no_tools_count}
                                onChange={e => setElasticPool({ ...elasticPool, no_tools_count: e.target.value })}
                                />
                            </div>
                            <div>
                                <label className="block text-sm font-medium mb-1.5">{t('accountManager.elasticPoolToolsOnlyCount')}</label>
                                <input
                                type="number"
                                min="0"
                                className="input-field"
                                value={elasticPool.tools_only_count}
                                onChange={e => setElasticPool({ ...elasticPool, tools_only_count: e.target.value })}
                                />
                            </div>
                        </>
                    )}

                    <p className="text-xs text-muted-foreground leading-relaxed">
                        {t('accountManager.elasticPoolDesc', { count: enabledCount })}
                    </p>

                    {error && (
                        <p className="text-sm text-red-500 leading-relaxed">
                            {error}
                        </p>
                    )}

                    <div className="flex justify-end gap-2 pt-2">
                        <button onClick={onClose} className="px-4 py-2 rounded-lg border border-border hover:bg-secondary transition-colors text-sm font-medium">{t('actions.cancel')}</button>
                        <button onClick={handleSave} disabled={loading} className="px-4 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors text-sm font-medium disabled:opacity-50">
                            {loading ? t('actions.saving') : t('actions.save')}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    )
}
