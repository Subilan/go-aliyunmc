-- 一次性数据回填：把旧版 tasks.output 写入 task_output_lines。
--
-- 旧版执行器每次 println 向 tasks.output 追加 "内容\n"；新版按行存放。
-- 这份历史数据不再拆分：整段文本作为 seq=1 的一行保存，保留全部换行。
-- GetTaskOutputString 会在这行后补一个 \n，因此重建结果与旧版完全一致。
--
-- 只处理已经结束的任务；发布期间仍在运行的任务会在后续 migrate 时补上。
-- 可重复执行：只处理还没有任何 task_output_lines 记录的任务。
INSERT INTO task_output_lines (task_id, seq, step, output, created_at)
SELECT
    t.id,
    1,
    COALESCE(t.step, 0),
    CASE
        WHEN RIGHT(t.output, 1) = CHAR(10)
            THEN LEFT(t.output, CHAR_LENGTH(t.output) - 1)
        ELSE t.output
    END,
    t.created_at
FROM tasks t
WHERE t.output <> ''
  AND t.status IN ('success', 'failed')
  AND NOT EXISTS (
      SELECT 1 FROM task_output_lines l WHERE l.task_id = t.id
  );
