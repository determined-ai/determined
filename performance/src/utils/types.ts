interface Authorization {
    token: string;
}

interface Experiment {
    id: string;
    metricName: string;
    metricType: string;
    batches: string;
    batchesMargin: string;
}

interface Task {
    id: string;
}

interface TestConfiguration {
    auth: Authorization;
    seededData: SeededData;
}

interface TestGroup {
    name: string;
    group: () => void;
    enabled?: boolean;
}

interface Trial {
    id: string;
}

interface Model {
    name: string;
    versionNum: string;
}

interface SeededData {
    model: Model;
    task: Task;
    trial: Trial;
    experiment: Experiment;
    workspace: Workspace;
}

interface Workspace {
    id: string;
    projectId: string;
}
