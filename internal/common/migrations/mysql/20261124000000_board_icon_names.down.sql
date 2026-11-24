-- 回滚是有损的：新的图标名无法映射回旧的四种语义图标，统一退回 discussion。
UPDATE `forum_boards`
  SET `icon` = 'discussion'
  WHERE `icon` NOT IN ('message-text', 'code', 'coffee', 'lightbulb');

UPDATE `forum_boards` SET `icon` = 'discussion' WHERE `icon` = 'message-text';
UPDATE `forum_boards` SET `icon` = 'life' WHERE `icon` = 'coffee';
UPDATE `forum_boards` SET `icon` = 'feedback' WHERE `icon` = 'lightbulb';

ALTER TABLE `forum_boards`
  MODIFY `icon` varchar(32) NOT NULL DEFAULT 'discussion';