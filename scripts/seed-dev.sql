-- Dev seed data — run after migrate-up
INSERT INTO rooms (slug, title, district, engine, difficulty, description) VALUES
  ('cache-thundering-herd', 'Cache Thundering Herd', 'Caching District', 'A', 'L1',
   'Your cache layer just went cold. Thousands of requests are slamming the database.'),
  ('dns-misconfiguration', 'DNS Misconfiguration', 'Networking District', 'B', 'L0',
   'Users report intermittent timeouts. Something is wrong with DNS resolution.'),
  ('kafka-consumer-lag', 'Kafka Consumer Lag', 'Eventing District', 'A', 'L2',
   'Consumer lag is growing. Messages are piling up and alerts are firing.')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO room_versions (room_id, version_number, status, changelog)
SELECT id, 1, 'PUBLISHED', 'Initial release'
FROM rooms
ON CONFLICT (room_id, version_number) DO NOTHING;
