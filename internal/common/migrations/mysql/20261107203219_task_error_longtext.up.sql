-- expand "tasks"."error" from varchar(191) to longtext for remote command diagnostics
ALTER TABLE `tasks` MODIFY COLUMN `error` longtext NULL;
