DROP TABLE IF EXISTS outbreaks;
DROP TABLE IF EXISTS diagnoses;

CREATE TABLE IF NOT EXISTS diagnosis_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    disease_id UUID,
    prediction VARCHAR(50),
    image_url VARCHAR(255) NOT NULL,
    confidence FLOAT NOT NULL,
    latitude FLOAT NOT NULL,
    longitude FLOAT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_diagnosis_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_diagnosis_disease FOREIGN KEY (disease_id) REFERENCES diseases(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS outbreaks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    disease_id UUID NOT NULL,
    diagnosis_id UUID,
    reported_by_user_id UUID,
    latitude FLOAT NOT NULL,
    longitude FLOAT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    is_verified BOOLEAN DEFAULT FALSE,
    verified_by UUID,
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_outbreak_disease FOREIGN KEY (disease_id) REFERENCES diseases(id) ON DELETE CASCADE,
    CONSTRAINT fk_outbreak_diagnosis FOREIGN KEY (diagnosis_id) REFERENCES diagnosis_history(id) ON DELETE SET NULL,
    CONSTRAINT fk_outbreak_reporter FOREIGN KEY (reported_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_outbreak_verifier FOREIGN KEY (verified_by) REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_diagnosis_history_user_id ON diagnosis_history(user_id);
CREATE INDEX IF NOT EXISTS idx_outbreaks_disease_id ON outbreaks(disease_id);
CREATE INDEX IF NOT EXISTS idx_outbreaks_location ON outbreaks(latitude, longitude);
