CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- USERS
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE, -- Nullable for OAuth users
    password_hash VARCHAR(255), -- Nullable for OAuth users
    role VARCHAR(20) DEFAULT 'FARMER' CHECK (role IN ('FARMER', 'EXPERT', 'ADMIN')),
    avatar_url TEXT,
    reset_token VARCHAR(255),
    reset_token_expires_at TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- USER IDENTITIES (Social Login Links)
CREATE TABLE user_identities (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider     VARCHAR(20) NOT NULL,    -- 'google' | 'line'
    provider_uid VARCHAR(255) NOT NULL,   -- Provider's unique user ID (sub)
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(provider, provider_uid)
);

-- NOTIFICATION SETTINGS
CREATE TABLE notification_settings (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    enabled BOOLEAN DEFAULT TRUE,
    radius_km DECIMAL(10, 2) DEFAULT 5.0,
    notify_high_severity BOOLEAN DEFAULT TRUE,
    notify_nearby BOOLEAN DEFAULT TRUE,
    latitude DECIMAL(9, 6),
    longitude DECIMAL(9, 6)
);

-- DISEASES LIBRARY
CREATE TABLE diseases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    alias VARCHAR NOT NULL UNIQUE, -- matches AI label (e.g., rice_blast)
    name VARCHAR NOT NULL,
    category VARCHAR NOT NULL,
    image_url VARCHAR, -- Nullable
    description TEXT NOT NULL,
    spread_details TEXT, -- Nullable
    match_weather JSONB DEFAULT '[]',
    symptoms JSONB DEFAULT '[]',
    prevention JSONB DEFAULT '[]',
    treatment JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- DIAGNOSIS HISTORY
CREATE TABLE diagnosis_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    disease_id UUID REFERENCES diseases(id) ON DELETE SET NULL,
    prediction VARCHAR(50),
    image_url VARCHAR(255) NOT NULL,
    confidence FLOAT NOT NULL,
    latitude FLOAT,
    longitude FLOAT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- OUTBREAKS tracking
CREATE TABLE outbreaks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    disease_id UUID NOT NULL REFERENCES diseases(id) ON DELETE CASCADE,
    diagnosis_id UUID REFERENCES diagnosis_history(id) ON DELETE SET NULL,
    reported_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    latitude FLOAT NOT NULL,
    longitude FLOAT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    is_verified BOOLEAN DEFAULT FALSE,
    verified_by UUID REFERENCES users(id) ON DELETE SET NULL,
    verified_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- COMMUNITY POSTS
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    image_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- COMMUNITY COMMENTS
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- COMMUNITY LIKES
CREATE TABLE likes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(post_id, user_id)
);

-- SYSTEM NOTIFICATIONS
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    type VARCHAR(50) NOT NULL,
    reference_id UUID,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- INDEXES
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_user_identities ON user_identities(provider, provider_uid);
CREATE INDEX idx_diseases_alias ON diseases(alias);
CREATE INDEX idx_diagnosis_history_user_id ON diagnosis_history(user_id);
CREATE INDEX idx_outbreaks_disease_id ON outbreaks(disease_id);
CREATE INDEX idx_outbreaks_location ON outbreaks(latitude, longitude);
CREATE INDEX idx_posts_user_id ON posts(user_id);
CREATE INDEX idx_comments_post_id ON comments(post_id);
CREATE INDEX idx_likes_post_id ON likes(post_id);
