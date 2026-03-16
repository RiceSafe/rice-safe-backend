-- Add location columns to notification_settings for localized alerts
ALTER TABLE notification_settings 
ADD COLUMN IF NOT EXISTS latitude DECIMAL(9, 6),
ADD COLUMN IF NOT EXISTS longitude DECIMAL(9, 6);
