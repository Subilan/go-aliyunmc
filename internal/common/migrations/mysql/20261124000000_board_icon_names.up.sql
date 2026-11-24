-- 图标改为 pixelarticons 的 kebab-case 名称，并为更长的图标名放宽列宽。
ALTER TABLE `forum_boards`
  MODIFY `icon` varchar(64) NOT NULL DEFAULT 'message-text';

-- 旧的四类语义图标映射为对应的 pixelarticons 图标。
UPDATE `forum_boards` SET `icon` = 'message-text' WHERE `icon` = 'discussion';
UPDATE `forum_boards` SET `icon` = 'coffee' WHERE `icon` = 'life';
UPDATE `forum_boards` SET `icon` = 'lightbulb' WHERE `icon` = 'feedback';