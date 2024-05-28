import Pivot from 'hew/Pivot';
import Notes from 'hew/RichTextEditor';
import { Loadable } from 'hew/utils/loadable';
import { string } from 'io-ts';
import _ from 'lodash';
import { useObservable } from 'micro-observables';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { unstable_useBlocker, useLocation, useNavigate, useParams } from 'react-router-dom';

import Page, { BreadCrumbRoute } from 'components/Page';
import { terminalRunStates } from 'constants/states';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import { SettingsConfig, useSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getExperimentDetails, patchExperiment } from 'services/api';
import workspaceStore from 'stores/workspaces';
import { ExperimentBase, Note, ValueOf, Workspace } from 'types';
import handleError, { ErrorLevel, ErrorType } from 'utils/error';
import { isAborted } from 'utils/service';

import ExperimentCodeViewer from './ExperimentDetails/ExperimentCodeViewer';
import ExperimentTrials from './ExperimentDetails/ExperimentTrials';

type Params = {
  searchId: string;
  tab?: ValueOf<typeof TabType>;
};

const TabType = {
  Code: 'code',
  Notes: 'notes',
  Trials: 'trials',
} as const;

const TAB_KEYS = Object.values(TabType);
const DEFAULT_TAB_KEY = TabType.Trials;

const SearchDetails: React.FC = () => {
  const { tab, searchId } = useParams<Params>();
  const [experiment, setExperiment] = useState<ExperimentBase>();
  const [pageError, setPageError] = useState<Error>();
  const canceler = useRef<AbortController>();
  const workspaces = Loadable.getOrElse([], useObservable(workspaceStore.workspaces));
  const id = parseInt(searchId ?? '');
  const pageRef = useRef<HTMLElement>(null);

  const navigate = useNavigate();
  const location = useLocation();
  const defaultTabKey = tab && TAB_KEYS.includes(tab) ? tab : DEFAULT_TAB_KEY;
  const [tabKey, setTabKey] = useState(defaultTabKey);
  const basePath = paths.searchDetails(id);

  const configForExperiment = (experimentId: number): SettingsConfig<{ filePath: string }> => ({
    settings: {
      filePath: {
        defaultValue: '',
        storageKey: 'filePath',
        type: string,
      },
    },
    storagePath: `selected-file-${experimentId}`,
  });
  const config: SettingsConfig<{ filePath: string }> = useMemo(() => {
    return configForExperiment(id);
  }, [id]);
  const { settings, updateSettings } = useSettings<{ filePath: string }>(config);
  const handleSelectFile = useCallback(
    (filePath: string) => {
      updateSettings({ filePath });
    },
    [updateSettings],
  );

  const { canModifyExperimentMetadata, canViewExperimentArtifacts } = usePermissions();
  const showExperimentArtifacts = experiment && canViewExperimentArtifacts({
    workspace: { id: experiment.workspaceId },
  });
  const editableNotes = experiment && canModifyExperimentMetadata({
    workspace: { id: experiment.workspaceId },
  });

  const handleTabChange = useCallback(
    (key: string) => {
      navigate(`${basePath}/${key}`, { replace: true });
    },
    [basePath, navigate],
  );

  useEffect(() => {
    setTabKey(tab ?? DEFAULT_TAB_KEY);
  }, [location.pathname, tab]);

  // Sets the default sub route.
  useEffect(() => {
    if (!tab || (tab && !TAB_KEYS.includes(tab))) {
      if (window.location.pathname.includes(basePath))
        navigate(`${basePath}/${tabKey}`, { replace: true });
    }
  }, [basePath, navigate, tab, tabKey]);

  const fetchExperimentDetails = useCallback(async () => {
    if (!searchId) return;
    try {
      const newExperiment = await getExperimentDetails(
        { id: parseInt(searchId) },
        { signal: canceler.current?.signal },
      );
      setExperiment((prevExperiment) =>
        _.isEqual(prevExperiment, newExperiment) ? prevExperiment : newExperiment,
      );
    } catch (e) {
      if (!pageError && !isAborted(e)) setPageError(e as Error);
    }
  }, [pageError, searchId]);

  const handleNotesUpdate = useCallback(
    async (notes: Note) => {
      const editedNotes = notes.contents;
      try {
        await patchExperiment({ body: { notes: editedNotes }, experimentId: id });
        await fetchExperimentDetails();
      } catch (e) {
        handleError(e, {
          level: ErrorLevel.Error,
          publicMessage: 'Please try again later.',
          publicSubject: 'Unable to update experiment notes.',
          silent: false,
          type: ErrorType.Server,
        });
      }
    },
    [id, fetchExperimentDetails],
  );

  const tabItems = [
    {
      children: (
        experiment && (
          <ExperimentTrials experiment={experiment} pageRef={pageRef} />
        )
      ),
      key: TabType.Trials,
      label: 'Trials',
    },
    {
      children: (
        experiment && showExperimentArtifacts && (
          <ExperimentCodeViewer
            experiment={experiment}
            selectedFilePath={settings.filePath}
            onSelectFile={handleSelectFile}
          />
        )
      ),
      key: TabType.Code,
      label: 'Code',
    },
    {
      children: (
        <Notes
          disabled={!editableNotes}
          disableTitle
          docs={{ contents: experiment?.notes ?? '', name: 'Notes' }}
          onError={handleError}
          onPageUnloadHook={unstable_useBlocker}
          onSave={handleNotesUpdate}
        />
      ),
      key: 'notes',
      label: 'Notes',
    },
  ];

  const { stopPolling } = usePolling(fetchExperimentDetails, { rerunOnNewFn: true });

  useEffect(() => {
    if (experiment && terminalRunStates.has(experiment.state)) {
      stopPolling();
    }
  }, [experiment, stopPolling]);

  const workspaceName = workspaces.find((ws: Workspace) => ws.id === experiment?.workspaceId)?.name;

  const pageBreadcrumb: BreadCrumbRoute[] = [
    workspaceName && experiment?.workspaceId !== 1
      ? {
        breadcrumbName: workspaceName,
        path: paths.workspaceDetails(experiment?.workspaceId ?? 1),
      }
      : {
        breadcrumbName: 'Uncategorized Experiments',
        path: paths.projectDetails(1),
      },
  ];

  if (experiment?.projectName && experiment?.projectId && experiment?.projectId !== 1)
    pageBreadcrumb.push({
      breadcrumbName: experiment?.projectName ?? '',
      path: paths.projectDetails(experiment?.projectId),
    });

  pageBreadcrumb.push({
    breadcrumbName: experiment?.name ?? '',
    path: paths.searchDetails(id),
  });

  return (
    <Page
      breadcrumb={pageBreadcrumb}
      stickyHeader
      title={`Search ${searchId}`}>
      <Pivot activeKey={tabKey} items={tabItems} onChange={handleTabChange} />
    </Page>
  );

};

export default SearchDetails;
