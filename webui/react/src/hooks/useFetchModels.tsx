import { ErrorType } from 'hew/utils/error';
import { Loadable, NotLoaded } from 'hew/utils/loadable';

import { useAsync } from 'hooks/useAsync';
import { getModels } from 'services/api';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';
import { ModelItem } from 'types';
import handleError from 'utils/error';
import { validateDetApiEnum } from 'utils/service';

export const useFetchModels = (modelsIn?: Loadable<ModelItem[]>): Loadable<ModelItem[]> => {
  return useAsync(
    async (canceler) => {
      if (modelsIn) return modelsIn;
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
        return response.models;
      } catch (e) {
        handleError(e, {
          publicSubject: 'Unable to fetch models.',
          silent: true,
          type: ErrorType.Api,
        });
        return NotLoaded;
      }
    },
    [modelsIn],
  );
};
