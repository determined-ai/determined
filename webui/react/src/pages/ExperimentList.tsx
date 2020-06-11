import React from 'react';

import ExperimentsTable from 'components/ExperimentTable';
import Page from 'components/Page';
import Experiments from 'contexts/Experiments';
import Users from 'contexts/Users';

const ExperimentList: React.FC = () => {
  const experiments = Experiments.useStateContext();
  const users = Users.useStateContext();
  // let experimentTasks: Task[] | undefined = experiments.data?.map(experimentToTask);
  // if (experimentTasks && (users.data !== undefined)) {
  //   experimentTasks = experimentTasks.map((exp: Task) => {
  //     const username = users.data?.find(user => user.id === exp.ownerId)?.username;
  //     return {
  //       ...exp,
  //       username,
  //     };
  //   });
  // }

  // TODO select and batch operation:
  // https://ant.design/components/table/#components-table-demo-row-selection-and-operation
  return (
    <Page title="Experiments">
      {
        <ExperimentsTable experiments={experiments.data} />
      }
    </Page>
  );
};

export default ExperimentList;
