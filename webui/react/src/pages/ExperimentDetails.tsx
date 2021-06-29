import React, { useCallback, useEffect, useState } from 'react';
import { useParams } from 'react-router';

import CreateExperimentModal, { CreateExperimentType } from 'components/CreateExperimentModal';
import Message, { MessageType } from 'components/Message';
import Page from 'components/Page';
import Spinner from 'components/Spinner';
import usePolling from 'hooks/usePolling';
import ExperimentDetailsHeader from 'pages/ExperimentDetails/ExperimentDetailsHeader';
import { paths, routeAll } from 'routes/utils';
import {
  getExperimentDetails, getExpTrials, getExpValidationHistory, isNotFound,
} from 'services/api';
import { createExperiment } from 'services/api';
import { isAborted } from 'services/utils';
import { ExperimentBase, ExperimentSearcherName, RawJson, ValidationHistory } from 'types';
import { clone, isEqual } from 'utils/data';
import { terminalRunStates, upgradeConfig } from 'utils/types';

import ExperimentMultiTrialTabs from './ExperimentDetails/ExperimentMultiTrialTabs';
import ExperimentSingleTrialTabs from './ExperimentDetails/ExperimentSingleTrialTabs';

interface Params {
  experimentId: string;
}

const ExperimentDetails: React.FC = () => {
  const { experimentId } = useParams<Params>();
  const [ canceler ] = useState(new AbortController());
  const [ experiment, setExperiment ] = useState<ExperimentBase>();
  const [ valHistory, setValHistory ] = useState<ValidationHistory[]>([]);
  const [ pageError, setPageError ] = useState<Error>();
  const [ firstTrialId, setFirstTrialId ] = useState<number>();
  const [ forkModalConfig, setForkModalConfig ] = useState<RawJson>();
  const [ forkModalError, setForkModalError ] = useState<string>();
  const [ isForkModalVisible, setIsForkModalVisible ] = useState(false);
  const [ isSingleTrial, setIsSingleTrial ] = useState(false);

  const id = parseInt(experimentId);

  const fetchExperimentDetails = useCallback(async () => {
    try {
      const [ experimentData, validationHistory ] = await Promise.all([
        getExperimentDetails({ id }, { signal: canceler.signal }),
        getExpValidationHistory({ id }, { signal: canceler.signal }),
      ]);
      if (!isEqual(experimentData, experiment)) setExperiment(experimentData);
      if (!isEqual(validationHistory, valHistory)) setValHistory(validationHistory);
      setIsSingleTrial(
        experimentData?.config.searcher.name === ExperimentSearcherName.Single
        || experimentData?.config.searcher.max_trials === 1,
      );
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e);
    }
  }, [
    experiment,
    id,
    canceler.signal,
    pageError,
    valHistory,
  ]);

  const { startPolling, stopPolling } = usePolling(fetchExperimentDetails);

  const showForkModal = useCallback((): void => {
    if (experiment?.configRaw) {
      const rawConfig: RawJson = clone(experiment.configRaw);
      if (rawConfig.description) rawConfig.description = `Fork of ${rawConfig.description}`;
      upgradeConfig(rawConfig);
      setForkModalConfig(rawConfig);
    }
    setIsForkModalVisible(true);
  }, [ experiment?.configRaw ]);

  const handleForkModalCancel = useCallback(() => {
    setIsForkModalVisible(false);
  }, []);

  const handleForkModalSubmit = useCallback(async (newConfig: string) => {
    try {
      const { id: configId } = await createExperiment({
        experimentConfig: newConfig,
        parentId: id,
      });

      // Reset experiment state and start polling for newly forked experiment.
      setIsForkModalVisible(false);
      setExperiment(undefined);

      // Route to newly forkex experiment.
      routeAll(paths.experimentDetails(configId));

      // Add a slight delay to allow polling function to update.
      setTimeout(() => startPolling(), 100);
    } catch (e) {
      setForkModalError(e.response?.data?.message || 'Unable to create experiment.');
      let errorMessage = 'Unable to fork experiment with the provided config.';
      if (e.name === 'YAMLException') {
        errorMessage = e.message;
      } else if (e.response?.data?.message) {
        errorMessage = e.response.data.message;
      }
      setForkModalError(errorMessage);
    }
  }, [ id, startPolling ]);

  useEffect(() => {
    if (experiment && terminalRunStates.has(experiment.state)) {
      stopPolling();
    }
  }, [ experiment, stopPolling ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

  useEffect(() => {
    (async () => {
      const expTrials = await getExpTrials({ id }, { signal: canceler.signal });
      if (expTrials.trials[0]) {
        setFirstTrialId(expTrials.trials[0].id);
      }
    })();
  }, [ canceler, id ]);

  if (isNaN(id)) {
    return <Message title={`Invalid Experiment ID ${experimentId}`} />;
  } else if (pageError) {
    const message = isNotFound(pageError) ?
      `Unable to find Experiment ${experimentId}` :
      `Unable to fetch Experiment ${experimentId}`;
    return <Message title={message} type={MessageType.Warning} />;
  } else if (!experiment) {
    return <Spinner />;
  }

  let tabsComponent = <ExperimentMultiTrialTabs experiment={experiment} />;

  if (isSingleTrial) {
    if (!firstTrialId) {
      return <Spinner />;
    }
    tabsComponent = <ExperimentSingleTrialTabs experiment={experiment} trialId={firstTrialId} />;
  }

  return (
    <Page
      headerComponent={<ExperimentDetailsHeader
        experiment={experiment}
        fetchExperimentDetails={fetchExperimentDetails}
        showForkModal={showForkModal}
      />}
      stickyHeader
      title={`Experiment ${experimentId}`}>
      {tabsComponent}
      <CreateExperimentModal
        config={forkModalConfig}
        error={forkModalError}
        title={`Fork Experiment ${id}`}
        type={CreateExperimentType.Fork}
        visible={isForkModalVisible}
        onCancel={handleForkModalCancel}
        onOk={handleForkModalSubmit}
      />
    </Page>
  );
};

export default ExperimentDetails;
