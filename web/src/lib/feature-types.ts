export interface TaskSchedule {
  id: number;
  name: string;
  enabled: boolean;
  schedule_type: string;
  timezone_mode: string;
  run_date?: string;
  end_date?: string;
  start_time: string;
  end_time?: string;
  weekdays?: number[];
  task_ids: number[];
  status: string;
  last_error?: string;
  next_start?: string;
  timezone?: string;
}
export interface FindingCase {
  id: string;
  title: string;
  reason: string;
  severity: string;
  count: number;
  version: number;
  report?: string;
  task_id?: string;
}

export interface FindingCaseRow {
  case?: FindingCase | null;
  finding?: { id: string; name: string; severity: string; task_id: string };
}
