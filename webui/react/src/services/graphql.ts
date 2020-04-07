import { RunState } from 'types';

interface GqlQuery {
  operationName?: string;
  query: string;
  variables?: Record<string, unknown>;
}

enum GqlOrderByType {
  Asc = 'asc',
  AscNullsFirst = 'asc_nulls_first',
  AscNullsLast = 'asc_nulls_last',
  Desc = 'desc',
  DescNullsFirst = 'desc_nulls_first',
  DescNullsLast = 'desc_nulls_last',
}

interface GqlBuildOptions {
  limit?: number;
  operationName?: string;
  orderBy?: Record<string, GqlOrderByType>;
  where?: string;
}

interface ExperimentListGqlBuildOptions {
  limit?: number;
  states?: RunState[];
}

export const buildGqlQuery = (
  table: string,
  fields: string[],
  options: GqlBuildOptions = {},
): GqlQuery => {
  // `operationName` is only required if there are multiple operations present in the query
  const operationName = options.operationName;
  const orderBy = options.orderBy ? JSON.stringify(options.orderBy).replace(/"/g, '') : null;
  const queryLimit = `limit: ${options.limit || 1000}`;
  const queryOrder = orderBy ? `order_by: ${orderBy}` : null;
  const queryWhere = options.where ? `where: ${options.where}` : null;
  const queryArgs = [ queryLimit, queryOrder, queryWhere ].filter(query => !!query).join(', ');
  const query = `
    query ${operationName || ''} {
      ${table} (${queryArgs}) {
        ${fields.join('\n')}
	    }
    }
  `;
  return { operationName, query };
};

export const buildExperimentListGqlQuery = (
  options: ExperimentListGqlBuildOptions = {},
): GqlQuery => {
  const fields = [
    'archived',
    'config',
    'end_time',
    'id',
    'owner_id',
    'parent_id',
    'progress',
    'start_time',
    'state',
  ];
  const where = options.states ?
    `{ state: { _in: ${JSON.stringify(options.states)} } }` :
    undefined;
  return buildGqlQuery('experiments', fields, {
    limit: options.limit,
    orderBy: { id: GqlOrderByType.Desc },
    where,
  });
};
