SET SCHEMA 'storage_console';

CREATE TYPE blob_property_source AS ENUM (
  'exif',
  'color'
);

CREATE TYPE blob_property_type AS ENUM (
  'Done',

-- exif properties
  'ApertureValue',
  'BrightnessValue',
  'ExposureBiasValue',
  'GPSAltitude',
  'Make',
  'Model',
  'Software',
  'DateTimeOriginal',
  'OffsetTimeOriginal',
  'ExposureTime',
  'ISOSpeedRatings',
  'LensModel',
  'GPSLatitude',
  'GPSLongitude',
  'FocalLengthIn35mmFilm',

-- color properties
  'ProminentColor1',
  'ProminentColor2',
  'ProminentColor3',
  'ColorCategory1',
  'ColorCategory2',
  'ColorCategory3'
);

CREATE TYPE blob_property_value_type AS ENUM (
  'Bool',
  'Fraction',
  'Integer',
  'Float',
  'Timestamp',
  'TimestampWithTimeZone',
  'Text'
);

CREATE TABLE IF NOT EXISTS blob_properties (
  id SERIAL PRIMARY KEY,
  blob_id INTEGER NOT NULL,
  source blob_property_source NOT NULL,
  property_type blob_property_type NOT NULL,
  value_type blob_property_value_type NOT NULL,

  value_bool BOOLEAN,
  value_numerator INTEGER,
  value_denominator INTEGER,
  value_text TEXT,
  value_integer INTEGER,
  value_float DOUBLE PRECISION,
  value_timestamp TIMESTAMP,
  value_timestamptz TIMESTAMPTZ,

  FOREIGN KEY (blob_id) REFERENCES blobs(id) ON DELETE CASCADE,

  CHECK (
    (value_type = 'Bool' AND value_bool IS NOT NULL AND value_numerator IS NULL AND value_denominator IS NULL AND value_text IS NULL AND value_integer IS NULL AND value_float IS NULL AND value_timestamp IS NULL AND value_timestamptz IS NULL) OR
    (value_type = 'Fraction' AND value_numerator IS NOT NULL AND value_denominator IS NOT NULL AND value_denominator != 0 AND value_text IS NULL AND value_integer IS NULL AND value_float IS NULL AND value_timestamp IS NULL AND value_timestamptz IS NULL) OR
    (value_type = 'Integer' AND value_integer IS NOT NULL AND value_numerator IS NULL AND value_denominator IS NULL AND value_text IS NULL AND value_float IS NULL AND value_timestamp IS NULL AND value_timestamptz IS NULL) OR
    (value_type = 'Float' AND value_float IS NOT NULL AND value_numerator IS NULL AND value_denominator IS NULL AND value_text IS NULL AND value_integer IS NULL AND value_timestamp IS NULL AND value_timestamptz IS NULL) OR
    (value_type = 'Timestamp' AND value_timestamp IS NOT NULL AND value_numerator IS NULL AND value_denominator IS NULL AND value_text IS NULL AND value_integer IS NULL AND value_float IS NULL AND value_timestamptz IS NULL) OR
    (value_type = 'TimestampWithTimeZone' AND value_timestamptz IS NOT NULL AND value_numerator IS NULL AND value_denominator IS NULL AND value_text IS NULL AND value_integer IS NULL AND value_float IS NULL AND value_timestamp IS NULL) OR
    (value_type = 'Text' AND value_text IS NOT NULL AND value_numerator IS NULL AND value_denominator IS NULL AND value_integer IS NULL AND value_float IS NULL AND value_timestamp IS NULL AND value_timestamptz IS NULL)
  )
);
