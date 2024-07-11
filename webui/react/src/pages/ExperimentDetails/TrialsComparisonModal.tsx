import Button from 'hew/Button';
import Icon from 'hew/Icon';
import Message from 'hew/Message';
import { Modal } from 'hew/Modal';
import Row from 'hew/Row';
import Select, { Option, SelectValue } from 'hew/Select';
import Spinner from 'hew/Spinner';
import { Label } from 'hew/Typography';
import { Loadable, NotLoaded } from 'hew/utils/loadable';
import usePrevious from 'hew/utils/usePrevious';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import HumanReadableNumber from 'components/HumanReadableNumber';
import Link from 'components/Link';
import MetricBadgeTag from 'components/MetricBadgeTag';
import MetricSelect from 'components/MetricSelect';
import { useAsync } from 'hooks/useAsync';
import useFeature from 'hooks/useFeature';
import useMetricNames from 'hooks/useMetricNames';
import { paths } from 'routes/utils';
import { getTrialDetails } from 'services/api';
import {
  BulkExperimentItem,
  FlatRun,
  Metric,
  MetricSummary,
  Primitive,
  TrialDetails,
  XOR,
} from 'types';
import { isNumber } from 'utils/data';
import handleError, { ErrorType } from 'utils/error';
import { isRun } from 'utils/run';
import { capitalize, humanReadableBytes, pluralizer } from 'utils/string';

import css from './TrialsComparisonModal.module.scss';

interface TablePropsBase {
  onUnselect?: (trialId: number) => void;
}

type TableProps = XOR<
  XOR<
    {
      trialIds: number[];
    },
    {
      trials: TrialDetails[];
    }
  > & { experiment: BulkExperimentItem | BulkExperimentItem[] },
  { runs: FlatRun[] }
> &
  TablePropsBase;

type ModalProps = TableProps & {
  onCancel?: () => void;
};

const TrialsComparisonModalComponent: React.FC<ModalProps> = ({
  onCancel,
  ...props
}: ModalProps) => {
  useEffect(() => {
    if (props.trialIds?.length === 0 || props.trials?.length === 0 || props.runs?.length === 0)
      onCancel?.();
  }, [onCancel, props.runs?.length, props.trialIds?.length, props.trials?.length]);
  const f_flat_runs = useFeature().isOn('flat_runs');
  const trialCopy = f_flat_runs ? 'Run' : 'Trial';

  return (
    <Modal
      submit={{
        handleError: () => {},
        handler: () => {},
        onComplete: onCancel,
        text: 'Close',
      }}
      title={
        props.experiment !== undefined && !Array.isArray(props.experiment)
          ? `${f_flat_runs ? 'Search' : 'Experiment'} ${props.experiment.id} ${trialCopy} Comparison`
          : `${trialCopy} Comparison'`
      }
      onClose={onCancel}>
      <TrialsComparisonTable {...props} />
    </Modal>
  );
};

export const TrialsComparisonTable: React.FC<TableProps> = ({
  trialIds,
  trials,
  experiment,
  runs,
  onUnselect,
}: TableProps) => {
  const [selectedHyperparameters, setSelectedHyperparameters] = useState<string[]>([]);
  const [selectedMetrics, setSelectedMetrics] = useState<Metric[]>([]);
  const colSpan = (Array.isArray(experiment) ? experiment.length : trialIds?.length ?? 0) + 1;
  const f_flat_runs = useFeature().isOn('flat_runs');

  // the loadable has the flat run type here to make the getOrElse later on succeed.
  const trialDetailsRequest = useAsync<TrialDetails[] | FlatRun[]>(
    async (canceler) => {
      if (trialIds === undefined) return NotLoaded;
      return (
        await Promise.all(
          trialIds.map(async (trialId: number) => {
            try {
              return await getTrialDetails({ id: trialId }, { signal: canceler.signal });
            } catch (e) {
              handleError(e);
              return undefined;
            }
          }),
        )
      ).filter((r): r is Exclude<typeof r, undefined> => !!r);
    },
    [trialIds],
  );

  const trialsDetails = useMemo(() => {
    return trialDetailsRequest.getOrElse(trials ?? runs ?? []);
  }, [trialDetailsRequest, trials, runs]);

  const handleTrialUnselect = useCallback((trialId: number) => onUnselect?.(trialId), [onUnselect]);

  const getCheckpointSize = useCallback((trial: TrialDetails | FlatRun) => {
    return humanReadableBytes(isRun(trial) ? trial.checkpointSize : trial.totalCheckpointSize);
  }, []);

  const totalCheckpointsSizes: Record<string, string> = useMemo(
    () =>
      Object.fromEntries(
        Object.values(trialsDetails).map((trial) => [trial.id, getCheckpointSize(trial)]),
      ),
    [getCheckpointSize, trialsDetails],
  );

  const experimentMap = useMemo(() => {
    return Array.isArray(experiment)
      ? experiment.reduce(
          (acc, cur) => ({ ...acc, [cur.id]: cur }),
          {} as Record<number, BulkExperimentItem>,
        )
      : experiment
        ? { [experiment.id]: experiment }
        : {};
  }, [experiment]);

  const experimentIds = useMemo(
    () =>
      Array.isArray(experiment)
        ? experiment.map((exp) => exp.id)
        : experiment
          ? [experiment.id]
          : [],
    [experiment],
  );

  const handleMetricNamesError = useCallback(
    (e: unknown) => {
      const entityName = f_flat_runs ? 'search' : 'experiment';
      handleError(e, {
        publicMessage: `Failed to load metric names for ${pluralizer(
          experimentIds.length,
          entityName,
          f_flat_runs ? 'searches' : undefined,
        )} ${experimentIds.join(', ')}.`,
        publicSubject: `${capitalize(entityName)} metric name stream failed.`,
        type: ErrorType.Api,
      });
    },
    [experimentIds, f_flat_runs],
  );

  const loadableMetrics = useMetricNames(experimentIds, handleMetricNamesError);
  const metrics: Metric[] = useMemo(() => {
    if (experiment) {
      return Loadable.getOrElse([], loadableMetrics);
    }
    return Object.values(
      runs.reduce(
        (comparisonMetrics, run) => {
          const runMetrics = Object.entries(run.summaryMetrics ?? {});
          runMetrics.forEach(([metricGroup, metrics]) => {
            Object.keys(metrics ?? {}).forEach(
              (metricName) =>
                (comparisonMetrics[`${metricGroup}.${metricName}`] = {
                  group: metricGroup,
                  name: metricName,
                }),
            );
          });
          return comparisonMetrics;
        },
        {} as Record<string, Metric>,
      ),
    );
  }, [experiment, loadableMetrics, runs]);

  const prevMetrics = usePrevious(metrics, []);

  useEffect(() => {
    setSelectedMetrics((prevSelectedMetrics) =>
      _.isEqual(prevSelectedMetrics, prevMetrics) ? metrics : prevSelectedMetrics,
    );
  }, [metrics, prevMetrics]);

  const onMetricSelect = useCallback((selectedMetrics: Metric[]) => {
    setSelectedMetrics(selectedMetrics);
  }, []);

  const latestMetrics = useMemo(
    () =>
      trialsDetails.reduce(
        (metricValues, trial) => {
          metricValues[trial.id] = Object.values<Record<string, MetricSummary> | null>(
            trial.summaryMetrics ?? {},
          ).reduce(
            (trialMetrics, curMetricType) => {
              for (const [metricName, metricSummary] of Object.entries<MetricSummary>(
                curMetricType ?? {},
              )) {
                if (metricSummary.last != null) trialMetrics[metricName] = metricSummary.last;
              }
              return trialMetrics;
            },
            {} as Record<string, Primitive>,
          );
          return metricValues;
        },
        {} as Record<number, Record<string, Primitive>>,
      ),
    [trialsDetails],
  );

  const hyperparameterNames = useMemo(() => {
    return [
      ...trialsDetails.reduce((hpSet, curTrial) => {
        Object.keys(curTrial.hyperparameters ?? {}).forEach((hp) => hpSet.add(hp));
        return hpSet;
      }, new Set<string>()),
    ];
  }, [trialsDetails]);

  const prevHps = usePrevious(hyperparameterNames, []);

  useEffect(() => {
    setSelectedHyperparameters((prevSelectedHps) =>
      _.isEqual(prevSelectedHps, prevHps) ? hyperparameterNames : prevSelectedHps,
    );
  }, [hyperparameterNames, prevHps]);

  const onHyperparameterSelect = useCallback((selectedHPs: SelectValue) => {
    setSelectedHyperparameters(selectedHPs as string[]);
  }, []);

  const isLoaded = useMemo(
    () => (trialIds ? trialsDetails.length === trialIds.length : true),
    [trialIds, trialsDetails],
  );

  return (
    <div className={css.base}>
      {!(trialsDetails.length === 0) ? (
        <Spinner center spinning={!isLoaded}>
          <table>
            <thead>
              <tr>
                <th />
                {trialsDetails.map((trial) => (
                  <th className={css.trialTag} key={trial.id}>
                    <Row justifyContent="space-between" width="fill">
                      <Label truncate={{ tooltip: true }}>
                        <Link
                          path={paths.trialDetails(
                            trial.id,
                            isRun(trial) ? undefined : trial.experimentId,
                          )}>
                          {!isRun(trial) && Array.isArray(experiment)
                            ? experimentMap[trial.experimentId]?.name
                            : `${isRun(trial) ? 'Run' : 'Trial'} ${trial.id}`}
                        </Link>
                      </Label>
                      {onUnselect ? (
                        <Button
                          icon={<Icon name="close" size="tiny" title="close" />}
                          size="small"
                          onClick={() => handleTrialUnselect(trial.id)}
                        />
                      ) : null}
                    </Row>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              <tr>
                <th scope="row">State</th>
                {trialsDetails.map((trial) => (
                  <td key={trial.id} style={{ textAlign: 'center' }}>
                    <Badge state={trial.state} type={BadgeType.State} />
                  </td>
                ))}
              </tr>
              {Array.isArray(experiment) && (
                <>
                  <tr>
                    <th scope="row">{f_flat_runs ? 'Search' : 'Experiment'} ID</th>
                    {trialsDetails.map((trial) => (
                      <td key={trial.id}>
                        <Label truncate={{ tooltip: true }}>
                          {isRun(trial) ? trial.experiment?.id : trial.experimentId}
                        </Label>
                      </td>
                    ))}
                  </tr>
                  <tr>
                    <th scope="row">{f_flat_runs ? 'Run' : 'Trial'} ID</th>
                    {trialsDetails.map((trial) => (
                      <td key={trial.id}>
                        <Label truncate={{ tooltip: true }}>{trial.id}</Label>
                      </td>
                    ))}
                  </tr>
                </>
              )}
              <tr>
                <th scope="row">Batches Processed</th>
                {trialsDetails.map((trial) => (
                  <td key={trial.id}>
                    <Label truncate={{ tooltip: true }}>
                      {!isRun(trial) && trial.totalBatchesProcessed}
                    </Label>
                  </td>
                ))}
              </tr>
              <tr>
                <th scope="row">Total Checkpoint Size</th>
                {trialsDetails.map((trial) => (
                  <td key={trial.id}>
                    <Label truncate={{ tooltip: true }}>{totalCheckpointsSizes[trial.id]}</Label>
                  </td>
                ))}
              </tr>
              <tr>
                <th className={css.tableSelectCell} colSpan={colSpan} scope="row">
                  <div className={css.tableSelectContainer}>
                    <MetricSelect
                      defaultMetrics={metrics}
                      metrics={metrics}
                      multiple
                      value={selectedMetrics}
                      onChange={onMetricSelect}
                    />
                  </div>
                </th>
              </tr>
              {selectedMetrics.map((metric) => (
                <tr key={`${metric.group}-${metric.name}`}>
                  <th scope="row">
                    <MetricBadgeTag metric={metric} />
                  </th>
                  {trialsDetails.map((trial) => {
                    const metricValue = latestMetrics[trial.id][metric.name];
                    return (
                      <td key={trial.id}>
                        {metricValue !== undefined ? (
                          typeof metricValue === 'number' ? (
                            <HumanReadableNumber num={metricValue} />
                          ) : (
                            <Label truncate={{ tooltip: true }}>{metricValue.toString()}</Label>
                          )
                        ) : null}
                      </td>
                    );
                  })}
                </tr>
              ))}
              <tr>
                <th className={css.tableSelectCell} colSpan={colSpan} scope="row">
                  <div className={css.tableSelectContainer}>
                    <Select
                      defaultValue={hyperparameterNames}
                      disableTags
                      label="Hyperparameters"
                      mode="multiple"
                      value={selectedHyperparameters}
                      width={200}
                      onChange={onHyperparameterSelect}>
                      {hyperparameterNames.map((hp) => (
                        <Option key={hp} value={hp}>
                          {hp}
                        </Option>
                      ))}
                    </Select>
                  </div>
                </th>
              </tr>
              {selectedHyperparameters.map((hp) => (
                <tr key={hp}>
                  <th scope="row">
                    <Label truncate={{ tooltip: true }}>{hp}</Label>
                  </th>
                  {trialsDetails.map((trial) => {
                    const hpValue = trial.hyperparameters?.[hp];
                    const stringValue = JSON.stringify(hpValue);
                    return (
                      <td key={trial.id}>
                        {isNumber(hpValue) ? (
                          <HumanReadableNumber num={hpValue} />
                        ) : (
                          <Label truncate={{ tooltip: true }}>{stringValue}</Label>
                        )}
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </Spinner>
      ) : (
        <Message title="No data available." />
      )}
    </div>
  );
};

export default TrialsComparisonModalComponent;
