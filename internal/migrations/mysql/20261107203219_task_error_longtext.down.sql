-- reverse: shrink "tasks"."error" to varchar(191)
ALTER TABLE `tasks` MODIFY COLUMN `error` varchar(191) NULL;
