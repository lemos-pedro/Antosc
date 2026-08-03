-- ==========================================
-- 005: DEPRECATED / NO-OP
-- ==========================================
-- A tabela assistant_audit_log já é criada em 003_assistant_audit.sql
-- com o esquema canónico usado pelo código Go:
--   role, question, tools_called, final_answer, grounded, ungrounded_values
--
-- Esta migração existia como rascunho alternativo (answer, tool_name, ...)
-- e conflitava com 003. Mantida como no-op para não partir sequências de
-- deploy que já referenciam o ficheiro 005.
-- ==========================================

SELECT 1;
