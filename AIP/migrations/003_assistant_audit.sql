-- ==========================================
-- 003: AUDITORIA DO ASSISTENTE
-- Regista cada pergunta, ferramentas chamadas e resposta, para permitir
-- verificar depois se alguma resposta não está apoiada em dados reais.
-- ==========================================

CREATE TABLE IF NOT EXISTS assistant_audit_log (

    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    role VARCHAR(50) NOT NULL,

    question TEXT NOT NULL,

    tools_called JSONB, -- lista de {name, arguments, result} de cada ferramenta chamada

    final_answer TEXT NOT NULL,

    grounded BOOLEAN NOT NULL DEFAULT TRUE, -- FALSE se a verificação detetou números não presentes nos dados

    ungrounded_values TEXT, -- valores suspeitos, se grounded = FALSE (para inspeção humana)

    created_at TIMESTAMP NOT NULL DEFAULT NOW()

);

CREATE INDEX IF NOT EXISTS idx_assistant_audit_created
ON assistant_audit_log(created_at);

CREATE INDEX IF NOT EXISTS idx_assistant_audit_grounded
ON assistant_audit_log(grounded);
