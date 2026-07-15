import { useState } from 'react'
import { CircleHelp, X } from 'lucide-react'

export default function ExpertPromptSegmentSection({ t, form, setForm }) {
    const [showTroubleshooting, setShowTroubleshooting] = useState(false)

    return (
        <>
            <div className="bg-card border border-border rounded-xl p-5 space-y-4">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                    <div className="space-y-1">
                        <h3 className="font-semibold">{t('settings.expertPromptSegmentTitle')}</h3>
                        <p className="text-sm text-muted-foreground">{t('settings.expertPromptSegmentDesc')}</p>
                    </div>
                    <button
                        type="button"
                        onClick={() => setShowTroubleshooting(true)}
                        className="inline-flex h-9 items-center gap-2 self-start rounded-lg border border-border bg-background px-3 text-sm font-medium text-muted-foreground hover:bg-secondary/70 hover:text-foreground"
                        title={t('settings.expertPromptSegmentTroubleshooting')}
                    >
                        <CircleHelp className="h-4 w-4" />
                        {t('settings.expertPromptSegmentTroubleshooting')}
                    </button>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <label className="flex items-start gap-3 rounded-lg border border-border bg-background/60 p-4">
                        <input
                            type="checkbox"
                            checked={Boolean(form.expert_prompt_segment?.enabled)}
                            onChange={(e) => setForm((prev) => ({
                                ...prev,
                                expert_prompt_segment: {
                                    ...prev.expert_prompt_segment,
                                    enabled: e.target.checked,
                                },
                            }))}
                            className="mt-1 h-4 w-4 rounded border-border"
                        />
                        <div className="space-y-1">
                            <span className="text-sm font-medium block">{t('settings.expertPromptSegmentEnabled')}</span>
                            <span className="text-xs text-muted-foreground block">{t('settings.expertPromptSegmentDesc')}</span>
                        </div>
                    </label>
                    <label className="text-sm space-y-2">
                        <span className="text-muted-foreground">{t('settings.expertPromptSegmentMaxChars')}</span>
                        <input
                            type="number"
                            min={1000}
                            max={100000000}
                            value={form.expert_prompt_segment?.max_chars ?? 80000}
                            onChange={(e) => setForm((prev) => ({
                                ...prev,
                                expert_prompt_segment: {
                                    ...prev.expert_prompt_segment,
                                    max_chars: Number(e.target.value || 0),
                                },
                            }))}
                            className="w-full bg-background border border-border rounded-lg px-3 py-2"
                        />
                        <p className="text-xs text-muted-foreground">{t('settings.expertPromptSegmentMaxCharsHelp')}</p>
                    </label>
                    <label className="text-sm space-y-2">
                        <span className="text-muted-foreground">{t('settings.expertPromptSegmentStopDelayMs')}</span>
                        <input
                            type="number"
                            min={0}
                            max={60000}
                            value={form.expert_prompt_segment?.stop_delay_ms ?? 1500}
                            onChange={(e) => setForm((prev) => ({
                                ...prev,
                                expert_prompt_segment: {
                                    ...prev.expert_prompt_segment,
                                    stop_delay_ms: Number(e.target.value || 0),
                                },
                            }))}
                            className="w-full bg-background border border-border rounded-lg px-3 py-2"
                        />
                        <p className="text-xs text-muted-foreground">{t('settings.expertPromptSegmentStopDelayMsHelp')}</p>
                    </label>
                </div>
            </div>

            {showTroubleshooting && (
                <div
                    className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 p-4 backdrop-blur-sm"
                    onClick={() => setShowTroubleshooting(false)}
                >
                    <div
                        className="w-full max-w-lg rounded-xl border border-border bg-card shadow-2xl"
                        onClick={(event) => event.stopPropagation()}
                    >
                        <div className="flex items-center justify-between gap-3 border-b border-border px-5 py-4">
                            <h3 className="font-semibold">{t('settings.expertPromptSegmentTroubleshootingTitle')}</h3>
                            <button
                                type="button"
                                onClick={() => setShowTroubleshooting(false)}
                                className="rounded-lg p-2 text-muted-foreground hover:bg-secondary/70 hover:text-foreground"
                                title={t('actions.cancel')}
                            >
                                <X className="h-4 w-4" />
                            </button>
                        </div>
                        <div className="px-5 py-4">
                            <p className="whitespace-pre-line text-sm leading-7 text-muted-foreground">
                                {t('settings.expertPromptSegmentTroubleshootingBody')}
                            </p>
                        </div>
                    </div>
                </div>
            )}
        </>
    )
}
