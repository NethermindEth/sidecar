{{- define "blockIndexerWorkerName" -}}
{{ .Values.workers.blockIndexer.name }}
{{- end -}}

{{- define "backfillServerName" -}}
{{ .Values.backfillServer.name }}
{{- end -}}

{{- define "transactionLogIndexerWorkerName" -}}
{{ .Values.workers.transactionLogIndexer.name }}
{{- end -}}

{{- define "contractIndexerWorkerName" -}}
{{ .Values.workers.contractIndexer.name }}
{{- end -}}

{{- define "restakedStrategiesWorkerName" -}}
{{ .Values.workers.restakedStrategies.name }}
{{- end -}}

{{- define "blockSubscriberName" -}}
{{ .Values.blockSubscriber.name }}
{{- end -}}

{{- define "blockLakeCommonConfig" -}}
{{ .Values.common.config.name }}
{{- end -}}

{{- define "formatEnvVarName" -}}
SIDECAR_{{ regexReplaceAll "\\." . "_" | upper }}
{{- end -}}
