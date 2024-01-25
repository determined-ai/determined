import { ErrorType } from 'hew/utils/error';
import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';
import { isEqual } from 'lodash';
import { useCallback, useEffect, useState } from 'react';

import { getModels } from 'services/api';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';
import { ModelItem } from 'types';
import handleError from 'utils/error';
import { validateDetApiEnum } from 'utils/service';

export const useFetchModels = (): Loadable<ModelItem[]> => {
  const [models, setModels] = useState<Loadable<ModelItem[]>>(NotLoaded);
  const [canceler] = useState(new AbortController());

  const fetchModels = useCallback(async () => {
    try {
      const response = await getModels(
        {
          archived: false,
          orderBy: 'ORDER_BY_DESC',
          sortBy: validateDetApiEnum(
            V1GetModelsRequestSortBy,
            V1GetModelsRequestSortBy.LASTUPDATEDTIME,
          ),
        },
        { signal: canceler.signal },
      );
      setModels((prev) => {
        const loadedModels = Loaded(response.models);
        if (isEqual(prev, loadedModels)) return prev;
        return loadedModels;
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to fetch models.',
        silent: true,
        type: ErrorType.Api,
      });
    }
  }, [canceler.signal]);

  useEffect(() => {
    fetchModels();
  }, [fetchModels]);

  return models;
};
