# Skip Log

Tests that require real LLM credentials are skipped when API keys are not available.
Set `DZ_LLM_API_KEY`, `DZ_LLM_BASE_URL`, `DZ_LLM_MODEL`, and `DZ_LLM_TYPE` environment variables, or configure API keys in `.dolphin/config.yaml`.

| Date | Test | Reason |
|------|------|--------|
| 2026-05-17 | TestLLMConfigLoading | No API key configured in config.yaml (api_key is empty) |
| 2026-05-17 | TestLLMProviderHealthCheck | Same — LLM not configured |
| 2026-05-17 | TestLLMProviderRoundTrip | Same — LLM not configured |
| 2026-05-17 | TestAgentCommandsReal | Same — LLM not configured |
| 2026-05-17 | TestAgentErrorRecoveryCtxCancel | Same — LLM not configured |
| 2026-05-17 | TestAgentWithRealProvider | Same — LLM not configured |
| 2026-05-17 | TestMultiProviderConfig | No providers section in config.yaml |
