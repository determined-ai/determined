import { Button, Col, Row, Table, Tooltip } from 'antd';
import { SorterResult } from 'antd/es/table/interface';
import { id } from 'fp-ts/lib/Reader';
import React, { useCallback, useMemo, useState } from 'react';

import CheckpointModal from 'components/CheckpointModal';
import HumanReadableFloat from 'components/HumanReadableFloat';
import Icon from 'components/Icon';
import Section from 'components/Section';
import { defaultRowClassName, getPaginationConfig, MINIMUM_PAGE_SIZE } from 'components/Table';
import useStorage from 'hooks/useStorage';
import ExperimentChart from 'pages/ExperimentDetails/ExperimentChart';
import ExperimentInfoBox from 'pages/ExperimentDetails/ExperimentInfoBox';
import { handlePath, paths } from 'routes/utils';
import { ApiSorter } from 'services/types';
import { CheckpointDetail, CheckpointWorkloadExtended, ExperimentBase, TrialItem, ValidationHistory } from 'types';
import { numericSorter } from 'utils/data';
import { getMetricValue } from 'utils/types';

import css from './ExperimentOverview.module.scss';
import { columns as defaultColumns } from './ExperimentOverview.table';

interface Props {
  experiment: ExperimentBase;
  onTagsChange: () => void;
  trials: TrialItem[];
  validationHistory: ValidationHistory[];
}

const STORAGE_PATH = 'experiment-detail';
const STORAGE_LIMIT_KEY = 'limit';
const STORAGE_SORTER_KEY = 'sorter';

const ExperimentOverview: React.FC<Props> = ({ experiment, trials, validationHistory, onTagsChange }: Props) => {
  const storage = useStorage(STORAGE_PATH);
  const initLimit = storage.getWithDefault(STORAGE_LIMIT_KEY, MINIMUM_PAGE_SIZE);
  const initSorter: ApiSorter | null = storage.get(STORAGE_SORTER_KEY);
  const [ pageSize, setPageSize ] = useState(initLimit);
  const [ sorter, setSorter ] = useState<ApiSorter | null>(initSorter);
  const [ activeCheckpoint, setActiveCheckpoint ] = useState<CheckpointWorkloadExtended>();
  const [ showCheckpoint, setShowCheckpoint ] = useState(false);

  const columns = useMemo(() => {
    const latestValidationRenderer = (_: string, record: TrialItem): React.ReactNode => {
      const value = getMetricValue(record.latestValidationMetric, metric);
      return value && <HumanReadableFloat num={value} />;
    };

    const latestValidationSorter = (a: TrialItem, b: TrialItem): number => {
      if (!metric) return 0;
      const aMetric = getMetricValue(a.latestValidationMetric, metric);
      const bMetric = getMetricValue(b.latestValidationMetric, metric);
      return numericSorter(aMetric, bMetric);
    };
    // const latestValidationRenderer = (_: string, record: TrialItem2): React.ReactNode => {
    //   const validationMetrics = record.latestValidationMetrics?.validationMetrics || {};
    //   const latestValidationMetric = metric && validationMetrics[metric];
    //   return latestValidationMetric &&
    //     <HumanReadableFloat num={latestValidationMetric} />;
    // };

    // const latestValidationSorter = (a: TrialItem, b: TrialItem): number => {
    //   if (!metric) return 0;
    //   const aMetric = a.latestValidationMetrics?.validationMetrics[metric];
    //   const bMetric = b.latestValidationMetrics?.validationMetrics[metric];
    //   return numericSorter(aMetric, bMetric);
    // };

    // const checkpointRenderer = (_: string, record: TrialItem): React.ReactNode => {
    //   if (!record.bestAvailableCheckpoint) return;
    //   const checkpoint: CheckpointDetail = {
    //     ...record.bestAvailableCheckpoint,
    //     batch: record.totalBatchesProcessed,
    //     experimentId: experiment.id,
    //     trialId: record.id,
    //   };
    //   return (
    //     <Tooltip title="View Checkpoint">
    //       <Button
    //         aria-label="View Checkpoint"
    //         icon={<Icon name="checkpoint" />}
    //         onClick={e => handleCheckpointShow(e, checkpoint)} />
    //     </Tooltip>
    //   );
    // };

    const checkpointRenderer = (_: string, record: TrialItem): React.ReactNode => {
      if (!record.bestAvailableCheckpoint) return;
      const checkpoint: CheckpointWorkloadExtended = {
        ...record.bestAvailableCheckpoint,
        experimentId: experiment.id,
        trialId: record.id,
      };
      return (
        <Tooltip title="View Checkpoint">
          <Button
            aria-label="View Checkpoint"
            icon={<Icon name="checkpoint" />}
            onClick={e => handleCheckpointShow(e, checkpoint)} />
        </Tooltip>
      );
    };

    const { metric, smallerIsBetter } = experiment.config?.searcher || {};
    const newColumns = [ ...defaultColumns ].map(column => {
      column.sortOrder = null;
      if (!sorter && column.key === 'bestValidation') {
        column.sortOrder = smallerIsBetter ? 'ascend' : 'descend';
      } else if (sorter && column.key === sorter.key) {
        column.sortOrder = sorter.descend ? 'descend' : 'ascend';
      }
      if (column.key === 'latestValidation') {
        column.render = latestValidationRenderer;
        column.sorter = latestValidationSorter;
      }
      if (column.key === 'checkpoint') column.render = checkpointRenderer;
      return column;
    });

    return newColumns;
  }, [ experiment.config, experiment.id, sorter ]);

  const handleTableChange = useCallback((tablePagination, tableFilters, sorter) => {
    if (Array.isArray(sorter)) return;

    const { columnKey, order } = sorter as SorterResult<TrialItem>;
    if (!columnKey || !columns.find(column => column.key === columnKey)) return;

    storage.set(STORAGE_SORTER_KEY, { descend: order === 'descend', key: columnKey as string });
    setSorter({ descend: order === 'descend', key: columnKey as string });

    storage.set(STORAGE_LIMIT_KEY, tablePagination.pageSize);
    setPageSize(tablePagination.pageSize);
  }, [ columns, setSorter, storage ]);

  const handleTableRow = useCallback((record: TrialItem) => {
    const handleClick = (event: React.MouseEvent) =>
      handlePath(event, { path: paths.trialDetails(record.id, experiment.id) });
    return { onAuxClick: handleClick, onClick: handleClick };
  }, [ id ]);

  const handleCheckpointShow = (
    event: React.MouseEvent,
    checkpoint: CheckpointWorkloadExtended,
  ) => {
    event.stopPropagation();
    setActiveCheckpoint(checkpoint);
    setShowCheckpoint(true);
  };
  // const handleCheckpointShow = (event: React.MouseEvent, checkpoint: CheckpointDetail) => {
  //   event.stopPropagation();
  //   setActiveCheckpoint(checkpoint);
  //   setShowCheckpoint(true);
  // };

  const handleCheckpointDismiss = useCallback(() => setShowCheckpoint(false), []);

  return (
    <div className={css.base}>
      <Row className={css.topRow} gutter={[ 16, 16 ]}>
        <Col lg={10} span={24} xl={8} xxl={6}>
          <ExperimentInfoBox
            experiment={experiment}
            onTagsChange={onTagsChange}
          />
        </Col>
        <Col lg={14} span={24} xl={16} xxl={18}>
          <ExperimentChart
            startTime={experiment.startTime}
            validationHistory={validationHistory}
            validationMetric={experiment.config?.searcher.metric} />
        </Col>
        <Col span={24}>
          <Section title="Trials">
            <Table
              columns={columns}
              dataSource={trials}
              pagination={getPaginationConfig(trials.length || 0, pageSize)}
              rowClassName={defaultRowClassName({ clickable: true })}
              rowKey="id"
              showSorterTooltip={false}
              size="small"
              onChange={handleTableChange}
              onRow={handleTableRow} />
          </Section>
        </Col>
      </Row>
      {activeCheckpoint && <CheckpointModal
        checkpoint={activeCheckpoint}
        config={experiment.config}
        show={showCheckpoint}
        title={`Best Checkpoint for Trial ${activeCheckpoint.trialId}`}
        onHide={handleCheckpointDismiss} />}
    </div>
  );
};

export default ExperimentOverview;
