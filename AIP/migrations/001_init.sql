-- +goose Up
-- ==========================================
-- AIP INITIAL DATABASE
-- Antosc Intelligence Platform
-- ==========================================


CREATE EXTENSION IF NOT EXISTS "uuid-ossp";


-- ==========================================
-- FEATURES
-- Dados preparados para Machine Learning
-- ==========================================

CREATE TABLE IF NOT EXISTS ai_features (

    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    tower_id UUID NOT NULL,

    feature_name VARCHAR(100) NOT NULL,

    feature_value DOUBLE PRECISION NOT NULL,

    unit VARCHAR(30),

    created_at TIMESTAMP NOT NULL DEFAULT NOW()

);



CREATE INDEX IF NOT EXISTS idx_ai_features_tower
ON ai_features(tower_id);



CREATE INDEX IF NOT EXISTS idx_ai_features_date
ON ai_features(created_at);



-- ==========================================
-- PREDICTIONS
-- Resultado dos modelos IA
-- ==========================================

CREATE TABLE IF NOT EXISTS ai_predictions (

    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    tower_id UUID NOT NULL,

    model_name VARCHAR(100) NOT NULL,

    score DOUBLE PRECISION NOT NULL,

    status VARCHAR(50),

    explanation TEXT,

    prediction_window INTEGER,

    created_at TIMESTAMP NOT NULL DEFAULT NOW()

);



CREATE INDEX IF NOT EXISTS idx_predictions_tower
ON ai_predictions(tower_id);



-- ==========================================
-- AI EVENTS
-- Eventos gerados pela inteligência
-- ==========================================

CREATE TABLE IF NOT EXISTS ai_events (

    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    tower_id UUID NOT NULL,

    event_type VARCHAR(100),

    severity VARCHAR(30),

    message TEXT,

    created_at TIMESTAMP NOT NULL DEFAULT NOW()

);



CREATE INDEX IF NOT EXISTS idx_ai_events_tower
ON ai_events(tower_id);



-- ==========================================
-- MODELS
-- Gestão dos modelos ML
-- ==========================================

CREATE TABLE IF NOT EXISTS ai_models (

    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    name VARCHAR(100) NOT NULL,

    version VARCHAR(50),

    algorithm VARCHAR(100),

    accuracy DOUBLE PRECISION,

    trained_at TIMESTAMP,

    active BOOLEAN DEFAULT TRUE

);

-- +goose Down
DROP TABLE IF EXISTS ai_models;
DROP TABLE IF EXISTS ai_events;
DROP TABLE IF EXISTS ai_predictions;
DROP TABLE IF EXISTS ai_features;
