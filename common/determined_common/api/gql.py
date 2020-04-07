import sgqlc.types

gql = sgqlc.types.Schema()


########################################################################
# Scalars and Enumerations
########################################################################
Boolean = sgqlc.types.Boolean

Float = sgqlc.types.Float

ID = sgqlc.types.ID

Int = sgqlc.types.Int

String = sgqlc.types.String


class agent_user_groups_select_column(sgqlc.types.Enum):
    __schema__ = gql
    __choices__ = ("gid", "group_", "id", "uid", "user_", "user_id")


class bytea(sgqlc.types.Scalar):
    __schema__ = gql


class checkpoint_state(sgqlc.types.Scalar):
    __schema__ = gql


class checkpoints_select_column(sgqlc.types.Enum):
    __schema__ = gql
    __choices__ = (
        "end_time",
        "id",
        "labels",
        "resources",
        "start_time",
        "state",
        "step_id",
        "trial_id",
        "uuid",
    )


class cluster_id_select_column(sgqlc.types.Enum):
    __schema__ = gql
    __choices__ = ("cluster_id",)


class config_files_select_column(sgqlc.types.Enum):
    __schema__ = gql
    __choices__ = ("content", "id")


class experiment_state(sgqlc.types.Scalar):
    __schema__ = gql


class experiments_select_column(sgqlc.types.Enum):
    __schema__ = gql
    __choices__ = (
        "archived",
        "config",
        "end_time",
        "git_commit",
        "git_commit_date",
        "git_committer",
        "git_remote",
        "id",
        "model_definition",
        "model_packages",
        "owner_id",
        "parent_id",
        "progress",
        "start_time",
        "state",
    )


class float8(sgqlc.types.Scalar):
    __schema__ = gql


class jsonb(sgqlc.types.Scalar):
    __schema__ = gql


class order_by(sgqlc.types.Enum):
    __schema__ = gql
    __choices__ = (
        "asc",
        "asc_nulls_first",
        "asc_nulls_last",
        "desc",
        "desc_nulls_first",
        "desc_nulls_last",
    )


class searcher_events_select_column(sgqlc.types.Enum):
    __schema__ = gql
    __choices__ = ("content", "event_type", "experiment_id", "id")


class step_state(sgqlc.types.Scalar):
    __schema__ = gql


class steps_select_column(sgqlc.types.Enum):
    __schema__ = gql
    __choices__ = ("end_time", "id", "metrics", "start_time", "state", "trial_id")


class templates_select_column(sgqlc.types.Enum):
    __schema__ = gql
    __choices__ = ("config", "name")


class timestamp(sgqlc.types.Scalar):
    __schema__ = gql


class timestamptz(sgqlc.types.Scalar):
    __schema__ = gql


class trial_logs_select_column(sgqlc.types.Enum):
    __schema__ = gql
    __choices__ = ("id", "message", "trial_id")


class trial_state(sgqlc.types.Scalar):
    __schema__ = gql


class trials_select_column(sgqlc.types.Enum):
    __schema__ = gql
    __choices__ = (
        "end_time",
        "experiment_id",
        "hparams",
        "id",
        "seed",
        "start_time",
        "state",
        "warm_start_checkpoint_id",
    )


class users_select_column(sgqlc.types.Enum):
    __schema__ = gql
    __choices__ = ("active", "admin", "id", "username")


class uuid(sgqlc.types.Scalar):
    __schema__ = gql


class validation_metrics_select_column(sgqlc.types.Enum):
    __schema__ = gql
    __choices__ = ("id", "raw", "signed")


class validation_state(sgqlc.types.Scalar):
    __schema__ = gql


class validations_select_column(sgqlc.types.Enum):
    __schema__ = gql
    __choices__ = ("end_time", "id", "metrics", "start_time", "state", "step_id", "trial_id")


########################################################################
# Input Objects
########################################################################
class Boolean_comparison_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_eq", "_gt", "_gte", "_in", "_is_null", "_lt", "_lte", "_neq", "_nin")
    _eq = sgqlc.types.Field(Boolean, graphql_name="_eq")
    _gt = sgqlc.types.Field(Boolean, graphql_name="_gt")
    _gte = sgqlc.types.Field(Boolean, graphql_name="_gte")
    _in = sgqlc.types.Field(sgqlc.types.list_of(sgqlc.types.non_null(Boolean)), graphql_name="_in")
    _is_null = sgqlc.types.Field(Boolean, graphql_name="_is_null")
    _lt = sgqlc.types.Field(Boolean, graphql_name="_lt")
    _lte = sgqlc.types.Field(Boolean, graphql_name="_lte")
    _neq = sgqlc.types.Field(Boolean, graphql_name="_neq")
    _nin = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(Boolean)), graphql_name="_nin"
    )


class Int_comparison_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_eq", "_gt", "_gte", "_in", "_is_null", "_lt", "_lte", "_neq", "_nin")
    _eq = sgqlc.types.Field(Int, graphql_name="_eq")
    _gt = sgqlc.types.Field(Int, graphql_name="_gt")
    _gte = sgqlc.types.Field(Int, graphql_name="_gte")
    _in = sgqlc.types.Field(sgqlc.types.list_of(sgqlc.types.non_null(Int)), graphql_name="_in")
    _is_null = sgqlc.types.Field(Boolean, graphql_name="_is_null")
    _lt = sgqlc.types.Field(Int, graphql_name="_lt")
    _lte = sgqlc.types.Field(Int, graphql_name="_lte")
    _neq = sgqlc.types.Field(Int, graphql_name="_neq")
    _nin = sgqlc.types.Field(sgqlc.types.list_of(sgqlc.types.non_null(Int)), graphql_name="_nin")


class String_comparison_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "_eq",
        "_gt",
        "_gte",
        "_ilike",
        "_in",
        "_is_null",
        "_like",
        "_lt",
        "_lte",
        "_neq",
        "_nilike",
        "_nin",
        "_nlike",
        "_nsimilar",
        "_similar",
    )
    _eq = sgqlc.types.Field(String, graphql_name="_eq")
    _gt = sgqlc.types.Field(String, graphql_name="_gt")
    _gte = sgqlc.types.Field(String, graphql_name="_gte")
    _ilike = sgqlc.types.Field(String, graphql_name="_ilike")
    _in = sgqlc.types.Field(sgqlc.types.list_of(sgqlc.types.non_null(String)), graphql_name="_in")
    _is_null = sgqlc.types.Field(Boolean, graphql_name="_is_null")
    _like = sgqlc.types.Field(String, graphql_name="_like")
    _lt = sgqlc.types.Field(String, graphql_name="_lt")
    _lte = sgqlc.types.Field(String, graphql_name="_lte")
    _neq = sgqlc.types.Field(String, graphql_name="_neq")
    _nilike = sgqlc.types.Field(String, graphql_name="_nilike")
    _nin = sgqlc.types.Field(sgqlc.types.list_of(sgqlc.types.non_null(String)), graphql_name="_nin")
    _nlike = sgqlc.types.Field(String, graphql_name="_nlike")
    _nsimilar = sgqlc.types.Field(String, graphql_name="_nsimilar")
    _similar = sgqlc.types.Field(String, graphql_name="_similar")


class agent_user_groups_aggregate_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("agent_user_groups_avg_order_by", graphql_name="avg")
    count = sgqlc.types.Field(order_by, graphql_name="count")
    max = sgqlc.types.Field("agent_user_groups_max_order_by", graphql_name="max")
    min = sgqlc.types.Field("agent_user_groups_min_order_by", graphql_name="min")
    stddev = sgqlc.types.Field("agent_user_groups_stddev_order_by", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field(
        "agent_user_groups_stddev_pop_order_by", graphql_name="stddev_pop"
    )
    stddev_samp = sgqlc.types.Field(
        "agent_user_groups_stddev_samp_order_by", graphql_name="stddev_samp"
    )
    sum = sgqlc.types.Field("agent_user_groups_sum_order_by", graphql_name="sum")
    var_pop = sgqlc.types.Field("agent_user_groups_var_pop_order_by", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("agent_user_groups_var_samp_order_by", graphql_name="var_samp")
    variance = sgqlc.types.Field("agent_user_groups_variance_order_by", graphql_name="variance")


class agent_user_groups_avg_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(order_by, graphql_name="gid")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    uid = sgqlc.types.Field(order_by, graphql_name="uid")
    user_id = sgqlc.types.Field(order_by, graphql_name="user_id")


class agent_user_groups_bool_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "_and",
        "_not",
        "_or",
        "gid",
        "group_",
        "id",
        "uid",
        "user",
        "user_",
        "user_id",
    )
    _and = sgqlc.types.Field(sgqlc.types.list_of("agent_user_groups_bool_exp"), graphql_name="_and")
    _not = sgqlc.types.Field("agent_user_groups_bool_exp", graphql_name="_not")
    _or = sgqlc.types.Field(sgqlc.types.list_of("agent_user_groups_bool_exp"), graphql_name="_or")
    gid = sgqlc.types.Field(Int_comparison_exp, graphql_name="gid")
    group_ = sgqlc.types.Field(String_comparison_exp, graphql_name="group_")
    id = sgqlc.types.Field(Int_comparison_exp, graphql_name="id")
    uid = sgqlc.types.Field(Int_comparison_exp, graphql_name="uid")
    user = sgqlc.types.Field("users_bool_exp", graphql_name="user")
    user_ = sgqlc.types.Field(String_comparison_exp, graphql_name="user_")
    user_id = sgqlc.types.Field(Int_comparison_exp, graphql_name="user_id")


class agent_user_groups_max_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("gid", "group_", "id", "uid", "user_", "user_id")
    gid = sgqlc.types.Field(order_by, graphql_name="gid")
    group_ = sgqlc.types.Field(order_by, graphql_name="group_")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    uid = sgqlc.types.Field(order_by, graphql_name="uid")
    user_ = sgqlc.types.Field(order_by, graphql_name="user_")
    user_id = sgqlc.types.Field(order_by, graphql_name="user_id")


class agent_user_groups_min_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("gid", "group_", "id", "uid", "user_", "user_id")
    gid = sgqlc.types.Field(order_by, graphql_name="gid")
    group_ = sgqlc.types.Field(order_by, graphql_name="group_")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    uid = sgqlc.types.Field(order_by, graphql_name="uid")
    user_ = sgqlc.types.Field(order_by, graphql_name="user_")
    user_id = sgqlc.types.Field(order_by, graphql_name="user_id")


class agent_user_groups_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("gid", "group_", "id", "uid", "user", "user_", "user_id")
    gid = sgqlc.types.Field(order_by, graphql_name="gid")
    group_ = sgqlc.types.Field(order_by, graphql_name="group_")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    uid = sgqlc.types.Field(order_by, graphql_name="uid")
    user = sgqlc.types.Field("users_order_by", graphql_name="user")
    user_ = sgqlc.types.Field(order_by, graphql_name="user_")
    user_id = sgqlc.types.Field(order_by, graphql_name="user_id")


class agent_user_groups_stddev_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(order_by, graphql_name="gid")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    uid = sgqlc.types.Field(order_by, graphql_name="uid")
    user_id = sgqlc.types.Field(order_by, graphql_name="user_id")


class agent_user_groups_stddev_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(order_by, graphql_name="gid")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    uid = sgqlc.types.Field(order_by, graphql_name="uid")
    user_id = sgqlc.types.Field(order_by, graphql_name="user_id")


class agent_user_groups_stddev_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(order_by, graphql_name="gid")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    uid = sgqlc.types.Field(order_by, graphql_name="uid")
    user_id = sgqlc.types.Field(order_by, graphql_name="user_id")


class agent_user_groups_sum_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(order_by, graphql_name="gid")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    uid = sgqlc.types.Field(order_by, graphql_name="uid")
    user_id = sgqlc.types.Field(order_by, graphql_name="user_id")


class agent_user_groups_var_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(order_by, graphql_name="gid")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    uid = sgqlc.types.Field(order_by, graphql_name="uid")
    user_id = sgqlc.types.Field(order_by, graphql_name="user_id")


class agent_user_groups_var_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(order_by, graphql_name="gid")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    uid = sgqlc.types.Field(order_by, graphql_name="uid")
    user_id = sgqlc.types.Field(order_by, graphql_name="user_id")


class agent_user_groups_variance_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(order_by, graphql_name="gid")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    uid = sgqlc.types.Field(order_by, graphql_name="uid")
    user_id = sgqlc.types.Field(order_by, graphql_name="user_id")


class best_checkpoint_by_metric_args(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("metric", "smaller_is_better", "tid")
    metric = sgqlc.types.Field(String, graphql_name="metric")
    smaller_is_better = sgqlc.types.Field(Boolean, graphql_name="smaller_is_better")
    tid = sgqlc.types.Field(Int, graphql_name="tid")


class bytea_comparison_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_eq", "_gt", "_gte", "_in", "_is_null", "_lt", "_lte", "_neq", "_nin")
    _eq = sgqlc.types.Field(bytea, graphql_name="_eq")
    _gt = sgqlc.types.Field(bytea, graphql_name="_gt")
    _gte = sgqlc.types.Field(bytea, graphql_name="_gte")
    _in = sgqlc.types.Field(sgqlc.types.list_of(sgqlc.types.non_null(bytea)), graphql_name="_in")
    _is_null = sgqlc.types.Field(Boolean, graphql_name="_is_null")
    _lt = sgqlc.types.Field(bytea, graphql_name="_lt")
    _lte = sgqlc.types.Field(bytea, graphql_name="_lte")
    _neq = sgqlc.types.Field(bytea, graphql_name="_neq")
    _nin = sgqlc.types.Field(sgqlc.types.list_of(sgqlc.types.non_null(bytea)), graphql_name="_nin")


class checkpoint_state_comparison_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_eq", "_gt", "_gte", "_in", "_is_null", "_lt", "_lte", "_neq", "_nin")
    _eq = sgqlc.types.Field(checkpoint_state, graphql_name="_eq")
    _gt = sgqlc.types.Field(checkpoint_state, graphql_name="_gt")
    _gte = sgqlc.types.Field(checkpoint_state, graphql_name="_gte")
    _in = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(checkpoint_state)), graphql_name="_in"
    )
    _is_null = sgqlc.types.Field(Boolean, graphql_name="_is_null")
    _lt = sgqlc.types.Field(checkpoint_state, graphql_name="_lt")
    _lte = sgqlc.types.Field(checkpoint_state, graphql_name="_lte")
    _neq = sgqlc.types.Field(checkpoint_state, graphql_name="_neq")
    _nin = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(checkpoint_state)), graphql_name="_nin"
    )


class checkpoints_aggregate_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("checkpoints_avg_order_by", graphql_name="avg")
    count = sgqlc.types.Field(order_by, graphql_name="count")
    max = sgqlc.types.Field("checkpoints_max_order_by", graphql_name="max")
    min = sgqlc.types.Field("checkpoints_min_order_by", graphql_name="min")
    stddev = sgqlc.types.Field("checkpoints_stddev_order_by", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("checkpoints_stddev_pop_order_by", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("checkpoints_stddev_samp_order_by", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("checkpoints_sum_order_by", graphql_name="sum")
    var_pop = sgqlc.types.Field("checkpoints_var_pop_order_by", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("checkpoints_var_samp_order_by", graphql_name="var_samp")
    variance = sgqlc.types.Field("checkpoints_variance_order_by", graphql_name="variance")


class checkpoints_avg_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class checkpoints_bool_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "_and",
        "_not",
        "_or",
        "end_time",
        "id",
        "labels",
        "resources",
        "start_time",
        "state",
        "step",
        "step_id",
        "trial_id",
        "trials",
        "uuid",
        "validation",
    )
    _and = sgqlc.types.Field(sgqlc.types.list_of("checkpoints_bool_exp"), graphql_name="_and")
    _not = sgqlc.types.Field("checkpoints_bool_exp", graphql_name="_not")
    _or = sgqlc.types.Field(sgqlc.types.list_of("checkpoints_bool_exp"), graphql_name="_or")
    end_time = sgqlc.types.Field("timestamptz_comparison_exp", graphql_name="end_time")
    id = sgqlc.types.Field(Int_comparison_exp, graphql_name="id")
    labels = sgqlc.types.Field("jsonb_comparison_exp", graphql_name="labels")
    resources = sgqlc.types.Field("jsonb_comparison_exp", graphql_name="resources")
    start_time = sgqlc.types.Field("timestamptz_comparison_exp", graphql_name="start_time")
    state = sgqlc.types.Field(checkpoint_state_comparison_exp, graphql_name="state")
    step = sgqlc.types.Field("steps_bool_exp", graphql_name="step")
    step_id = sgqlc.types.Field(Int_comparison_exp, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Int_comparison_exp, graphql_name="trial_id")
    trials = sgqlc.types.Field("trials_bool_exp", graphql_name="trials")
    uuid = sgqlc.types.Field("uuid_comparison_exp", graphql_name="uuid")
    validation = sgqlc.types.Field("validations_bool_exp", graphql_name="validation")


class checkpoints_max_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("end_time", "id", "start_time", "step_id", "trial_id")
    end_time = sgqlc.types.Field(order_by, graphql_name="end_time")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    start_time = sgqlc.types.Field(order_by, graphql_name="start_time")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class checkpoints_min_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("end_time", "id", "start_time", "step_id", "trial_id")
    end_time = sgqlc.types.Field(order_by, graphql_name="end_time")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    start_time = sgqlc.types.Field(order_by, graphql_name="start_time")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class checkpoints_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "end_time",
        "id",
        "labels",
        "resources",
        "start_time",
        "state",
        "step",
        "step_id",
        "trial_id",
        "trials_aggregate",
        "uuid",
        "validation",
    )
    end_time = sgqlc.types.Field(order_by, graphql_name="end_time")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    labels = sgqlc.types.Field(order_by, graphql_name="labels")
    resources = sgqlc.types.Field(order_by, graphql_name="resources")
    start_time = sgqlc.types.Field(order_by, graphql_name="start_time")
    state = sgqlc.types.Field(order_by, graphql_name="state")
    step = sgqlc.types.Field("steps_order_by", graphql_name="step")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")
    trials_aggregate = sgqlc.types.Field(
        "trials_aggregate_order_by", graphql_name="trials_aggregate"
    )
    uuid = sgqlc.types.Field(order_by, graphql_name="uuid")
    validation = sgqlc.types.Field("validations_order_by", graphql_name="validation")


class checkpoints_stddev_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class checkpoints_stddev_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class checkpoints_stddev_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class checkpoints_sum_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class checkpoints_var_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class checkpoints_var_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class checkpoints_variance_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class cluster_id_aggregate_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("count", "max", "min")
    count = sgqlc.types.Field(order_by, graphql_name="count")
    max = sgqlc.types.Field("cluster_id_max_order_by", graphql_name="max")
    min = sgqlc.types.Field("cluster_id_min_order_by", graphql_name="min")


class cluster_id_bool_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_and", "_not", "_or", "cluster_id")
    _and = sgqlc.types.Field(sgqlc.types.list_of("cluster_id_bool_exp"), graphql_name="_and")
    _not = sgqlc.types.Field("cluster_id_bool_exp", graphql_name="_not")
    _or = sgqlc.types.Field(sgqlc.types.list_of("cluster_id_bool_exp"), graphql_name="_or")
    cluster_id = sgqlc.types.Field(String_comparison_exp, graphql_name="cluster_id")


class cluster_id_max_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("cluster_id",)
    cluster_id = sgqlc.types.Field(order_by, graphql_name="cluster_id")


class cluster_id_min_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("cluster_id",)
    cluster_id = sgqlc.types.Field(order_by, graphql_name="cluster_id")


class cluster_id_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("cluster_id",)
    cluster_id = sgqlc.types.Field(order_by, graphql_name="cluster_id")


class config_files_aggregate_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("config_files_avg_order_by", graphql_name="avg")
    count = sgqlc.types.Field(order_by, graphql_name="count")
    max = sgqlc.types.Field("config_files_max_order_by", graphql_name="max")
    min = sgqlc.types.Field("config_files_min_order_by", graphql_name="min")
    stddev = sgqlc.types.Field("config_files_stddev_order_by", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("config_files_stddev_pop_order_by", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("config_files_stddev_samp_order_by", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("config_files_sum_order_by", graphql_name="sum")
    var_pop = sgqlc.types.Field("config_files_var_pop_order_by", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("config_files_var_samp_order_by", graphql_name="var_samp")
    variance = sgqlc.types.Field("config_files_variance_order_by", graphql_name="variance")


class config_files_avg_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class config_files_bool_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_and", "_not", "_or", "content", "id")
    _and = sgqlc.types.Field(sgqlc.types.list_of("config_files_bool_exp"), graphql_name="_and")
    _not = sgqlc.types.Field("config_files_bool_exp", graphql_name="_not")
    _or = sgqlc.types.Field(sgqlc.types.list_of("config_files_bool_exp"), graphql_name="_or")
    content = sgqlc.types.Field(bytea_comparison_exp, graphql_name="content")
    id = sgqlc.types.Field(Int_comparison_exp, graphql_name="id")


class config_files_max_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class config_files_min_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class config_files_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("content", "id")
    content = sgqlc.types.Field(order_by, graphql_name="content")
    id = sgqlc.types.Field(order_by, graphql_name="id")


class config_files_stddev_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class config_files_stddev_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class config_files_stddev_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class config_files_sum_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class config_files_var_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class config_files_var_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class config_files_variance_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class experiment_state_comparison_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_eq", "_gt", "_gte", "_in", "_is_null", "_lt", "_lte", "_neq", "_nin")
    _eq = sgqlc.types.Field(experiment_state, graphql_name="_eq")
    _gt = sgqlc.types.Field(experiment_state, graphql_name="_gt")
    _gte = sgqlc.types.Field(experiment_state, graphql_name="_gte")
    _in = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(experiment_state)), graphql_name="_in"
    )
    _is_null = sgqlc.types.Field(Boolean, graphql_name="_is_null")
    _lt = sgqlc.types.Field(experiment_state, graphql_name="_lt")
    _lte = sgqlc.types.Field(experiment_state, graphql_name="_lte")
    _neq = sgqlc.types.Field(experiment_state, graphql_name="_neq")
    _nin = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(experiment_state)), graphql_name="_nin"
    )


class experiments_aggregate_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("experiments_avg_order_by", graphql_name="avg")
    count = sgqlc.types.Field(order_by, graphql_name="count")
    max = sgqlc.types.Field("experiments_max_order_by", graphql_name="max")
    min = sgqlc.types.Field("experiments_min_order_by", graphql_name="min")
    stddev = sgqlc.types.Field("experiments_stddev_order_by", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("experiments_stddev_pop_order_by", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("experiments_stddev_samp_order_by", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("experiments_sum_order_by", graphql_name="sum")
    var_pop = sgqlc.types.Field("experiments_var_pop_order_by", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("experiments_var_samp_order_by", graphql_name="var_samp")
    variance = sgqlc.types.Field("experiments_variance_order_by", graphql_name="variance")


class experiments_avg_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    owner_id = sgqlc.types.Field(order_by, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(order_by, graphql_name="parent_id")
    progress = sgqlc.types.Field(order_by, graphql_name="progress")


class experiments_bool_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "_and",
        "_not",
        "_or",
        "archived",
        "config",
        "end_time",
        "git_commit",
        "git_commit_date",
        "git_committer",
        "git_remote",
        "id",
        "model_definition",
        "model_packages",
        "owner",
        "owner_id",
        "parent_id",
        "progress",
        "searcher_events",
        "start_time",
        "state",
        "trials",
    )
    _and = sgqlc.types.Field(sgqlc.types.list_of("experiments_bool_exp"), graphql_name="_and")
    _not = sgqlc.types.Field("experiments_bool_exp", graphql_name="_not")
    _or = sgqlc.types.Field(sgqlc.types.list_of("experiments_bool_exp"), graphql_name="_or")
    archived = sgqlc.types.Field(Boolean_comparison_exp, graphql_name="archived")
    config = sgqlc.types.Field("jsonb_comparison_exp", graphql_name="config")
    end_time = sgqlc.types.Field("timestamptz_comparison_exp", graphql_name="end_time")
    git_commit = sgqlc.types.Field(String_comparison_exp, graphql_name="git_commit")
    git_commit_date = sgqlc.types.Field("timestamp_comparison_exp", graphql_name="git_commit_date")
    git_committer = sgqlc.types.Field(String_comparison_exp, graphql_name="git_committer")
    git_remote = sgqlc.types.Field(String_comparison_exp, graphql_name="git_remote")
    id = sgqlc.types.Field(Int_comparison_exp, graphql_name="id")
    model_definition = sgqlc.types.Field(bytea_comparison_exp, graphql_name="model_definition")
    model_packages = sgqlc.types.Field(bytea_comparison_exp, graphql_name="model_packages")
    owner = sgqlc.types.Field("users_bool_exp", graphql_name="owner")
    owner_id = sgqlc.types.Field(Int_comparison_exp, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(Int_comparison_exp, graphql_name="parent_id")
    progress = sgqlc.types.Field("float8_comparison_exp", graphql_name="progress")
    searcher_events = sgqlc.types.Field("searcher_events_bool_exp", graphql_name="searcher_events")
    start_time = sgqlc.types.Field("timestamptz_comparison_exp", graphql_name="start_time")
    state = sgqlc.types.Field(experiment_state_comparison_exp, graphql_name="state")
    trials = sgqlc.types.Field("trials_bool_exp", graphql_name="trials")


class experiments_max_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "end_time",
        "git_commit",
        "git_committer",
        "git_remote",
        "id",
        "owner_id",
        "parent_id",
        "progress",
        "start_time",
    )
    end_time = sgqlc.types.Field(order_by, graphql_name="end_time")
    git_commit = sgqlc.types.Field(order_by, graphql_name="git_commit")
    git_committer = sgqlc.types.Field(order_by, graphql_name="git_committer")
    git_remote = sgqlc.types.Field(order_by, graphql_name="git_remote")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    owner_id = sgqlc.types.Field(order_by, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(order_by, graphql_name="parent_id")
    progress = sgqlc.types.Field(order_by, graphql_name="progress")
    start_time = sgqlc.types.Field(order_by, graphql_name="start_time")


class experiments_min_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "end_time",
        "git_commit",
        "git_committer",
        "git_remote",
        "id",
        "owner_id",
        "parent_id",
        "progress",
        "start_time",
    )
    end_time = sgqlc.types.Field(order_by, graphql_name="end_time")
    git_commit = sgqlc.types.Field(order_by, graphql_name="git_commit")
    git_committer = sgqlc.types.Field(order_by, graphql_name="git_committer")
    git_remote = sgqlc.types.Field(order_by, graphql_name="git_remote")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    owner_id = sgqlc.types.Field(order_by, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(order_by, graphql_name="parent_id")
    progress = sgqlc.types.Field(order_by, graphql_name="progress")
    start_time = sgqlc.types.Field(order_by, graphql_name="start_time")


class experiments_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "archived",
        "config",
        "end_time",
        "git_commit",
        "git_commit_date",
        "git_committer",
        "git_remote",
        "id",
        "model_definition",
        "model_packages",
        "owner",
        "owner_id",
        "parent_id",
        "progress",
        "searcher_events_aggregate",
        "start_time",
        "state",
        "trials_aggregate",
    )
    archived = sgqlc.types.Field(order_by, graphql_name="archived")
    config = sgqlc.types.Field(order_by, graphql_name="config")
    end_time = sgqlc.types.Field(order_by, graphql_name="end_time")
    git_commit = sgqlc.types.Field(order_by, graphql_name="git_commit")
    git_commit_date = sgqlc.types.Field(order_by, graphql_name="git_commit_date")
    git_committer = sgqlc.types.Field(order_by, graphql_name="git_committer")
    git_remote = sgqlc.types.Field(order_by, graphql_name="git_remote")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    model_definition = sgqlc.types.Field(order_by, graphql_name="model_definition")
    model_packages = sgqlc.types.Field(order_by, graphql_name="model_packages")
    owner = sgqlc.types.Field("users_order_by", graphql_name="owner")
    owner_id = sgqlc.types.Field(order_by, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(order_by, graphql_name="parent_id")
    progress = sgqlc.types.Field(order_by, graphql_name="progress")
    searcher_events_aggregate = sgqlc.types.Field(
        "searcher_events_aggregate_order_by", graphql_name="searcher_events_aggregate"
    )
    start_time = sgqlc.types.Field(order_by, graphql_name="start_time")
    state = sgqlc.types.Field(order_by, graphql_name="state")
    trials_aggregate = sgqlc.types.Field(
        "trials_aggregate_order_by", graphql_name="trials_aggregate"
    )


class experiments_stddev_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    owner_id = sgqlc.types.Field(order_by, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(order_by, graphql_name="parent_id")
    progress = sgqlc.types.Field(order_by, graphql_name="progress")


class experiments_stddev_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    owner_id = sgqlc.types.Field(order_by, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(order_by, graphql_name="parent_id")
    progress = sgqlc.types.Field(order_by, graphql_name="progress")


class experiments_stddev_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    owner_id = sgqlc.types.Field(order_by, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(order_by, graphql_name="parent_id")
    progress = sgqlc.types.Field(order_by, graphql_name="progress")


class experiments_sum_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    owner_id = sgqlc.types.Field(order_by, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(order_by, graphql_name="parent_id")
    progress = sgqlc.types.Field(order_by, graphql_name="progress")


class experiments_var_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    owner_id = sgqlc.types.Field(order_by, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(order_by, graphql_name="parent_id")
    progress = sgqlc.types.Field(order_by, graphql_name="progress")


class experiments_var_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    owner_id = sgqlc.types.Field(order_by, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(order_by, graphql_name="parent_id")
    progress = sgqlc.types.Field(order_by, graphql_name="progress")


class experiments_variance_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    owner_id = sgqlc.types.Field(order_by, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(order_by, graphql_name="parent_id")
    progress = sgqlc.types.Field(order_by, graphql_name="progress")


class float8_comparison_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_eq", "_gt", "_gte", "_in", "_is_null", "_lt", "_lte", "_neq", "_nin")
    _eq = sgqlc.types.Field(float8, graphql_name="_eq")
    _gt = sgqlc.types.Field(float8, graphql_name="_gt")
    _gte = sgqlc.types.Field(float8, graphql_name="_gte")
    _in = sgqlc.types.Field(sgqlc.types.list_of(sgqlc.types.non_null(float8)), graphql_name="_in")
    _is_null = sgqlc.types.Field(Boolean, graphql_name="_is_null")
    _lt = sgqlc.types.Field(float8, graphql_name="_lt")
    _lte = sgqlc.types.Field(float8, graphql_name="_lte")
    _neq = sgqlc.types.Field(float8, graphql_name="_neq")
    _nin = sgqlc.types.Field(sgqlc.types.list_of(sgqlc.types.non_null(float8)), graphql_name="_nin")


class jsonb_comparison_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "_contained_in",
        "_contains",
        "_eq",
        "_gt",
        "_gte",
        "_has_key",
        "_has_keys_all",
        "_has_keys_any",
        "_in",
        "_is_null",
        "_lt",
        "_lte",
        "_neq",
        "_nin",
    )
    _contained_in = sgqlc.types.Field(jsonb, graphql_name="_contained_in")
    _contains = sgqlc.types.Field(jsonb, graphql_name="_contains")
    _eq = sgqlc.types.Field(jsonb, graphql_name="_eq")
    _gt = sgqlc.types.Field(jsonb, graphql_name="_gt")
    _gte = sgqlc.types.Field(jsonb, graphql_name="_gte")
    _has_key = sgqlc.types.Field(String, graphql_name="_has_key")
    _has_keys_all = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(String)), graphql_name="_has_keys_all"
    )
    _has_keys_any = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(String)), graphql_name="_has_keys_any"
    )
    _in = sgqlc.types.Field(sgqlc.types.list_of(sgqlc.types.non_null(jsonb)), graphql_name="_in")
    _is_null = sgqlc.types.Field(Boolean, graphql_name="_is_null")
    _lt = sgqlc.types.Field(jsonb, graphql_name="_lt")
    _lte = sgqlc.types.Field(jsonb, graphql_name="_lte")
    _neq = sgqlc.types.Field(jsonb, graphql_name="_neq")
    _nin = sgqlc.types.Field(sgqlc.types.list_of(sgqlc.types.non_null(jsonb)), graphql_name="_nin")


class searcher_events_aggregate_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("searcher_events_avg_order_by", graphql_name="avg")
    count = sgqlc.types.Field(order_by, graphql_name="count")
    max = sgqlc.types.Field("searcher_events_max_order_by", graphql_name="max")
    min = sgqlc.types.Field("searcher_events_min_order_by", graphql_name="min")
    stddev = sgqlc.types.Field("searcher_events_stddev_order_by", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("searcher_events_stddev_pop_order_by", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field(
        "searcher_events_stddev_samp_order_by", graphql_name="stddev_samp"
    )
    sum = sgqlc.types.Field("searcher_events_sum_order_by", graphql_name="sum")
    var_pop = sgqlc.types.Field("searcher_events_var_pop_order_by", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("searcher_events_var_samp_order_by", graphql_name="var_samp")
    variance = sgqlc.types.Field("searcher_events_variance_order_by", graphql_name="variance")


class searcher_events_avg_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")


class searcher_events_bool_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "_and",
        "_not",
        "_or",
        "content",
        "event_type",
        "experiment",
        "experiment_id",
        "id",
    )
    _and = sgqlc.types.Field(sgqlc.types.list_of("searcher_events_bool_exp"), graphql_name="_and")
    _not = sgqlc.types.Field("searcher_events_bool_exp", graphql_name="_not")
    _or = sgqlc.types.Field(sgqlc.types.list_of("searcher_events_bool_exp"), graphql_name="_or")
    content = sgqlc.types.Field(jsonb_comparison_exp, graphql_name="content")
    event_type = sgqlc.types.Field(String_comparison_exp, graphql_name="event_type")
    experiment = sgqlc.types.Field(experiments_bool_exp, graphql_name="experiment")
    experiment_id = sgqlc.types.Field(Int_comparison_exp, graphql_name="experiment_id")
    id = sgqlc.types.Field(Int_comparison_exp, graphql_name="id")


class searcher_events_max_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("event_type", "experiment_id", "id")
    event_type = sgqlc.types.Field(order_by, graphql_name="event_type")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")


class searcher_events_min_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("event_type", "experiment_id", "id")
    event_type = sgqlc.types.Field(order_by, graphql_name="event_type")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")


class searcher_events_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("content", "event_type", "experiment", "experiment_id", "id")
    content = sgqlc.types.Field(order_by, graphql_name="content")
    event_type = sgqlc.types.Field(order_by, graphql_name="event_type")
    experiment = sgqlc.types.Field(experiments_order_by, graphql_name="experiment")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")


class searcher_events_stddev_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")


class searcher_events_stddev_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")


class searcher_events_stddev_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")


class searcher_events_sum_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")


class searcher_events_var_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")


class searcher_events_var_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")


class searcher_events_variance_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")


class step_state_comparison_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_eq", "_gt", "_gte", "_in", "_is_null", "_lt", "_lte", "_neq", "_nin")
    _eq = sgqlc.types.Field(step_state, graphql_name="_eq")
    _gt = sgqlc.types.Field(step_state, graphql_name="_gt")
    _gte = sgqlc.types.Field(step_state, graphql_name="_gte")
    _in = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(step_state)), graphql_name="_in"
    )
    _is_null = sgqlc.types.Field(Boolean, graphql_name="_is_null")
    _lt = sgqlc.types.Field(step_state, graphql_name="_lt")
    _lte = sgqlc.types.Field(step_state, graphql_name="_lte")
    _neq = sgqlc.types.Field(step_state, graphql_name="_neq")
    _nin = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(step_state)), graphql_name="_nin"
    )


class steps_aggregate_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("steps_avg_order_by", graphql_name="avg")
    count = sgqlc.types.Field(order_by, graphql_name="count")
    max = sgqlc.types.Field("steps_max_order_by", graphql_name="max")
    min = sgqlc.types.Field("steps_min_order_by", graphql_name="min")
    stddev = sgqlc.types.Field("steps_stddev_order_by", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("steps_stddev_pop_order_by", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("steps_stddev_samp_order_by", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("steps_sum_order_by", graphql_name="sum")
    var_pop = sgqlc.types.Field("steps_var_pop_order_by", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("steps_var_samp_order_by", graphql_name="var_samp")
    variance = sgqlc.types.Field("steps_variance_order_by", graphql_name="variance")


class steps_avg_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class steps_bool_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "_and",
        "_not",
        "_or",
        "checkpoint",
        "end_time",
        "id",
        "metrics",
        "start_time",
        "state",
        "trial",
        "trial_id",
        "validation",
    )
    _and = sgqlc.types.Field(sgqlc.types.list_of("steps_bool_exp"), graphql_name="_and")
    _not = sgqlc.types.Field("steps_bool_exp", graphql_name="_not")
    _or = sgqlc.types.Field(sgqlc.types.list_of("steps_bool_exp"), graphql_name="_or")
    checkpoint = sgqlc.types.Field(checkpoints_bool_exp, graphql_name="checkpoint")
    end_time = sgqlc.types.Field("timestamptz_comparison_exp", graphql_name="end_time")
    id = sgqlc.types.Field(Int_comparison_exp, graphql_name="id")
    metrics = sgqlc.types.Field(jsonb_comparison_exp, graphql_name="metrics")
    start_time = sgqlc.types.Field("timestamptz_comparison_exp", graphql_name="start_time")
    state = sgqlc.types.Field(step_state_comparison_exp, graphql_name="state")
    trial = sgqlc.types.Field("trials_bool_exp", graphql_name="trial")
    trial_id = sgqlc.types.Field(Int_comparison_exp, graphql_name="trial_id")
    validation = sgqlc.types.Field("validations_bool_exp", graphql_name="validation")


class steps_max_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("end_time", "id", "start_time", "trial_id")
    end_time = sgqlc.types.Field(order_by, graphql_name="end_time")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    start_time = sgqlc.types.Field(order_by, graphql_name="start_time")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class steps_min_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("end_time", "id", "start_time", "trial_id")
    end_time = sgqlc.types.Field(order_by, graphql_name="end_time")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    start_time = sgqlc.types.Field(order_by, graphql_name="start_time")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class steps_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "checkpoint",
        "end_time",
        "id",
        "metrics",
        "start_time",
        "state",
        "trial",
        "trial_id",
        "validation",
    )
    checkpoint = sgqlc.types.Field(checkpoints_order_by, graphql_name="checkpoint")
    end_time = sgqlc.types.Field(order_by, graphql_name="end_time")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    metrics = sgqlc.types.Field(order_by, graphql_name="metrics")
    start_time = sgqlc.types.Field(order_by, graphql_name="start_time")
    state = sgqlc.types.Field(order_by, graphql_name="state")
    trial = sgqlc.types.Field("trials_order_by", graphql_name="trial")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")
    validation = sgqlc.types.Field("validations_order_by", graphql_name="validation")


class steps_stddev_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class steps_stddev_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class steps_stddev_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class steps_sum_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class steps_var_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class steps_var_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class steps_variance_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class templates_aggregate_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("count", "max", "min")
    count = sgqlc.types.Field(order_by, graphql_name="count")
    max = sgqlc.types.Field("templates_max_order_by", graphql_name="max")
    min = sgqlc.types.Field("templates_min_order_by", graphql_name="min")


class templates_bool_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_and", "_not", "_or", "config", "name")
    _and = sgqlc.types.Field(sgqlc.types.list_of("templates_bool_exp"), graphql_name="_and")
    _not = sgqlc.types.Field("templates_bool_exp", graphql_name="_not")
    _or = sgqlc.types.Field(sgqlc.types.list_of("templates_bool_exp"), graphql_name="_or")
    config = sgqlc.types.Field(jsonb_comparison_exp, graphql_name="config")
    name = sgqlc.types.Field(String_comparison_exp, graphql_name="name")


class templates_max_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("name",)
    name = sgqlc.types.Field(order_by, graphql_name="name")


class templates_min_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("name",)
    name = sgqlc.types.Field(order_by, graphql_name="name")


class templates_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("config", "name")
    config = sgqlc.types.Field(order_by, graphql_name="config")
    name = sgqlc.types.Field(order_by, graphql_name="name")


class timestamp_comparison_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_eq", "_gt", "_gte", "_in", "_is_null", "_lt", "_lte", "_neq", "_nin")
    _eq = sgqlc.types.Field(timestamp, graphql_name="_eq")
    _gt = sgqlc.types.Field(timestamp, graphql_name="_gt")
    _gte = sgqlc.types.Field(timestamp, graphql_name="_gte")
    _in = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(timestamp)), graphql_name="_in"
    )
    _is_null = sgqlc.types.Field(Boolean, graphql_name="_is_null")
    _lt = sgqlc.types.Field(timestamp, graphql_name="_lt")
    _lte = sgqlc.types.Field(timestamp, graphql_name="_lte")
    _neq = sgqlc.types.Field(timestamp, graphql_name="_neq")
    _nin = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(timestamp)), graphql_name="_nin"
    )


class timestamptz_comparison_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_eq", "_gt", "_gte", "_in", "_is_null", "_lt", "_lte", "_neq", "_nin")
    _eq = sgqlc.types.Field(timestamptz, graphql_name="_eq")
    _gt = sgqlc.types.Field(timestamptz, graphql_name="_gt")
    _gte = sgqlc.types.Field(timestamptz, graphql_name="_gte")
    _in = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(timestamptz)), graphql_name="_in"
    )
    _is_null = sgqlc.types.Field(Boolean, graphql_name="_is_null")
    _lt = sgqlc.types.Field(timestamptz, graphql_name="_lt")
    _lte = sgqlc.types.Field(timestamptz, graphql_name="_lte")
    _neq = sgqlc.types.Field(timestamptz, graphql_name="_neq")
    _nin = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(timestamptz)), graphql_name="_nin"
    )


class trial_logs_aggregate_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("trial_logs_avg_order_by", graphql_name="avg")
    count = sgqlc.types.Field(order_by, graphql_name="count")
    max = sgqlc.types.Field("trial_logs_max_order_by", graphql_name="max")
    min = sgqlc.types.Field("trial_logs_min_order_by", graphql_name="min")
    stddev = sgqlc.types.Field("trial_logs_stddev_order_by", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("trial_logs_stddev_pop_order_by", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("trial_logs_stddev_samp_order_by", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("trial_logs_sum_order_by", graphql_name="sum")
    var_pop = sgqlc.types.Field("trial_logs_var_pop_order_by", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("trial_logs_var_samp_order_by", graphql_name="var_samp")
    variance = sgqlc.types.Field("trial_logs_variance_order_by", graphql_name="variance")


class trial_logs_avg_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class trial_logs_bool_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_and", "_not", "_or", "id", "message", "trial", "trial_id")
    _and = sgqlc.types.Field(sgqlc.types.list_of("trial_logs_bool_exp"), graphql_name="_and")
    _not = sgqlc.types.Field("trial_logs_bool_exp", graphql_name="_not")
    _or = sgqlc.types.Field(sgqlc.types.list_of("trial_logs_bool_exp"), graphql_name="_or")
    id = sgqlc.types.Field(Int_comparison_exp, graphql_name="id")
    message = sgqlc.types.Field(bytea_comparison_exp, graphql_name="message")
    trial = sgqlc.types.Field("trials_bool_exp", graphql_name="trial")
    trial_id = sgqlc.types.Field(Int_comparison_exp, graphql_name="trial_id")


class trial_logs_max_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class trial_logs_min_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class trial_logs_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "message", "trial", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    message = sgqlc.types.Field(order_by, graphql_name="message")
    trial = sgqlc.types.Field("trials_order_by", graphql_name="trial")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class trial_logs_stddev_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class trial_logs_stddev_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class trial_logs_stddev_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class trial_logs_sum_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class trial_logs_var_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class trial_logs_var_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class trial_logs_variance_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class trial_state_comparison_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_eq", "_gt", "_gte", "_in", "_is_null", "_lt", "_lte", "_neq", "_nin")
    _eq = sgqlc.types.Field(trial_state, graphql_name="_eq")
    _gt = sgqlc.types.Field(trial_state, graphql_name="_gt")
    _gte = sgqlc.types.Field(trial_state, graphql_name="_gte")
    _in = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(trial_state)), graphql_name="_in"
    )
    _is_null = sgqlc.types.Field(Boolean, graphql_name="_is_null")
    _lt = sgqlc.types.Field(trial_state, graphql_name="_lt")
    _lte = sgqlc.types.Field(trial_state, graphql_name="_lte")
    _neq = sgqlc.types.Field(trial_state, graphql_name="_neq")
    _nin = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(trial_state)), graphql_name="_nin"
    )


class trials_aggregate_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("trials_avg_order_by", graphql_name="avg")
    count = sgqlc.types.Field(order_by, graphql_name="count")
    max = sgqlc.types.Field("trials_max_order_by", graphql_name="max")
    min = sgqlc.types.Field("trials_min_order_by", graphql_name="min")
    stddev = sgqlc.types.Field("trials_stddev_order_by", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("trials_stddev_pop_order_by", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("trials_stddev_samp_order_by", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("trials_sum_order_by", graphql_name="sum")
    var_pop = sgqlc.types.Field("trials_var_pop_order_by", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("trials_var_samp_order_by", graphql_name="var_samp")
    variance = sgqlc.types.Field("trials_variance_order_by", graphql_name="variance")


class trials_avg_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    seed = sgqlc.types.Field(order_by, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(order_by, graphql_name="warm_start_checkpoint_id")


class trials_bool_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "_and",
        "_not",
        "_or",
        "checkpoint",
        "checkpoints",
        "end_time",
        "experiment",
        "experiment_id",
        "hparams",
        "id",
        "seed",
        "start_time",
        "state",
        "steps",
        "trial_logs",
        "validations",
        "warm_start_checkpoint_id",
    )
    _and = sgqlc.types.Field(sgqlc.types.list_of("trials_bool_exp"), graphql_name="_and")
    _not = sgqlc.types.Field("trials_bool_exp", graphql_name="_not")
    _or = sgqlc.types.Field(sgqlc.types.list_of("trials_bool_exp"), graphql_name="_or")
    checkpoint = sgqlc.types.Field(checkpoints_bool_exp, graphql_name="checkpoint")
    checkpoints = sgqlc.types.Field(checkpoints_bool_exp, graphql_name="checkpoints")
    end_time = sgqlc.types.Field(timestamptz_comparison_exp, graphql_name="end_time")
    experiment = sgqlc.types.Field(experiments_bool_exp, graphql_name="experiment")
    experiment_id = sgqlc.types.Field(Int_comparison_exp, graphql_name="experiment_id")
    hparams = sgqlc.types.Field(jsonb_comparison_exp, graphql_name="hparams")
    id = sgqlc.types.Field(Int_comparison_exp, graphql_name="id")
    seed = sgqlc.types.Field(Int_comparison_exp, graphql_name="seed")
    start_time = sgqlc.types.Field(timestamptz_comparison_exp, graphql_name="start_time")
    state = sgqlc.types.Field(trial_state_comparison_exp, graphql_name="state")
    steps = sgqlc.types.Field(steps_bool_exp, graphql_name="steps")
    trial_logs = sgqlc.types.Field(trial_logs_bool_exp, graphql_name="trial_logs")
    validations = sgqlc.types.Field("validations_bool_exp", graphql_name="validations")
    warm_start_checkpoint_id = sgqlc.types.Field(
        Int_comparison_exp, graphql_name="warm_start_checkpoint_id"
    )


class trials_max_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "end_time",
        "experiment_id",
        "id",
        "seed",
        "start_time",
        "warm_start_checkpoint_id",
    )
    end_time = sgqlc.types.Field(order_by, graphql_name="end_time")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    seed = sgqlc.types.Field(order_by, graphql_name="seed")
    start_time = sgqlc.types.Field(order_by, graphql_name="start_time")
    warm_start_checkpoint_id = sgqlc.types.Field(order_by, graphql_name="warm_start_checkpoint_id")


class trials_min_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "end_time",
        "experiment_id",
        "id",
        "seed",
        "start_time",
        "warm_start_checkpoint_id",
    )
    end_time = sgqlc.types.Field(order_by, graphql_name="end_time")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    seed = sgqlc.types.Field(order_by, graphql_name="seed")
    start_time = sgqlc.types.Field(order_by, graphql_name="start_time")
    warm_start_checkpoint_id = sgqlc.types.Field(order_by, graphql_name="warm_start_checkpoint_id")


class trials_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "checkpoint",
        "checkpoints_aggregate",
        "end_time",
        "experiment",
        "experiment_id",
        "hparams",
        "id",
        "seed",
        "start_time",
        "state",
        "steps_aggregate",
        "trial_logs_aggregate",
        "validations_aggregate",
        "warm_start_checkpoint_id",
    )
    checkpoint = sgqlc.types.Field(checkpoints_order_by, graphql_name="checkpoint")
    checkpoints_aggregate = sgqlc.types.Field(
        checkpoints_aggregate_order_by, graphql_name="checkpoints_aggregate"
    )
    end_time = sgqlc.types.Field(order_by, graphql_name="end_time")
    experiment = sgqlc.types.Field(experiments_order_by, graphql_name="experiment")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    hparams = sgqlc.types.Field(order_by, graphql_name="hparams")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    seed = sgqlc.types.Field(order_by, graphql_name="seed")
    start_time = sgqlc.types.Field(order_by, graphql_name="start_time")
    state = sgqlc.types.Field(order_by, graphql_name="state")
    steps_aggregate = sgqlc.types.Field(steps_aggregate_order_by, graphql_name="steps_aggregate")
    trial_logs_aggregate = sgqlc.types.Field(
        trial_logs_aggregate_order_by, graphql_name="trial_logs_aggregate"
    )
    validations_aggregate = sgqlc.types.Field(
        "validations_aggregate_order_by", graphql_name="validations_aggregate"
    )
    warm_start_checkpoint_id = sgqlc.types.Field(order_by, graphql_name="warm_start_checkpoint_id")


class trials_stddev_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    seed = sgqlc.types.Field(order_by, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(order_by, graphql_name="warm_start_checkpoint_id")


class trials_stddev_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    seed = sgqlc.types.Field(order_by, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(order_by, graphql_name="warm_start_checkpoint_id")


class trials_stddev_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    seed = sgqlc.types.Field(order_by, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(order_by, graphql_name="warm_start_checkpoint_id")


class trials_sum_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    seed = sgqlc.types.Field(order_by, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(order_by, graphql_name="warm_start_checkpoint_id")


class trials_var_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    seed = sgqlc.types.Field(order_by, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(order_by, graphql_name="warm_start_checkpoint_id")


class trials_var_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    seed = sgqlc.types.Field(order_by, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(order_by, graphql_name="warm_start_checkpoint_id")


class trials_variance_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(order_by, graphql_name="experiment_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    seed = sgqlc.types.Field(order_by, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(order_by, graphql_name="warm_start_checkpoint_id")


class users_aggregate_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("users_avg_order_by", graphql_name="avg")
    count = sgqlc.types.Field(order_by, graphql_name="count")
    max = sgqlc.types.Field("users_max_order_by", graphql_name="max")
    min = sgqlc.types.Field("users_min_order_by", graphql_name="min")
    stddev = sgqlc.types.Field("users_stddev_order_by", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("users_stddev_pop_order_by", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("users_stddev_samp_order_by", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("users_sum_order_by", graphql_name="sum")
    var_pop = sgqlc.types.Field("users_var_pop_order_by", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("users_var_samp_order_by", graphql_name="var_samp")
    variance = sgqlc.types.Field("users_variance_order_by", graphql_name="variance")


class users_avg_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class users_bool_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "_and",
        "_not",
        "_or",
        "active",
        "admin",
        "agent_user_group",
        "experiments",
        "id",
        "username",
    )
    _and = sgqlc.types.Field(sgqlc.types.list_of("users_bool_exp"), graphql_name="_and")
    _not = sgqlc.types.Field("users_bool_exp", graphql_name="_not")
    _or = sgqlc.types.Field(sgqlc.types.list_of("users_bool_exp"), graphql_name="_or")
    active = sgqlc.types.Field(Boolean_comparison_exp, graphql_name="active")
    admin = sgqlc.types.Field(Boolean_comparison_exp, graphql_name="admin")
    agent_user_group = sgqlc.types.Field(
        agent_user_groups_bool_exp, graphql_name="agent_user_group"
    )
    experiments = sgqlc.types.Field(experiments_bool_exp, graphql_name="experiments")
    id = sgqlc.types.Field(Int_comparison_exp, graphql_name="id")
    username = sgqlc.types.Field(String_comparison_exp, graphql_name="username")


class users_max_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "username")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    username = sgqlc.types.Field(order_by, graphql_name="username")


class users_min_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "username")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    username = sgqlc.types.Field(order_by, graphql_name="username")


class users_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "active",
        "admin",
        "agent_user_group",
        "experiments_aggregate",
        "id",
        "username",
    )
    active = sgqlc.types.Field(order_by, graphql_name="active")
    admin = sgqlc.types.Field(order_by, graphql_name="admin")
    agent_user_group = sgqlc.types.Field(
        agent_user_groups_order_by, graphql_name="agent_user_group"
    )
    experiments_aggregate = sgqlc.types.Field(
        experiments_aggregate_order_by, graphql_name="experiments_aggregate"
    )
    id = sgqlc.types.Field(order_by, graphql_name="id")
    username = sgqlc.types.Field(order_by, graphql_name="username")


class users_stddev_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class users_stddev_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class users_stddev_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class users_sum_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class users_var_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class users_var_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class users_variance_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(order_by, graphql_name="id")


class uuid_comparison_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_eq", "_gt", "_gte", "_in", "_is_null", "_lt", "_lte", "_neq", "_nin")
    _eq = sgqlc.types.Field(uuid, graphql_name="_eq")
    _gt = sgqlc.types.Field(uuid, graphql_name="_gt")
    _gte = sgqlc.types.Field(uuid, graphql_name="_gte")
    _in = sgqlc.types.Field(sgqlc.types.list_of(sgqlc.types.non_null(uuid)), graphql_name="_in")
    _is_null = sgqlc.types.Field(Boolean, graphql_name="_is_null")
    _lt = sgqlc.types.Field(uuid, graphql_name="_lt")
    _lte = sgqlc.types.Field(uuid, graphql_name="_lte")
    _neq = sgqlc.types.Field(uuid, graphql_name="_neq")
    _nin = sgqlc.types.Field(sgqlc.types.list_of(sgqlc.types.non_null(uuid)), graphql_name="_nin")


class validation_metrics_aggregate_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("validation_metrics_avg_order_by", graphql_name="avg")
    count = sgqlc.types.Field(order_by, graphql_name="count")
    max = sgqlc.types.Field("validation_metrics_max_order_by", graphql_name="max")
    min = sgqlc.types.Field("validation_metrics_min_order_by", graphql_name="min")
    stddev = sgqlc.types.Field("validation_metrics_stddev_order_by", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field(
        "validation_metrics_stddev_pop_order_by", graphql_name="stddev_pop"
    )
    stddev_samp = sgqlc.types.Field(
        "validation_metrics_stddev_samp_order_by", graphql_name="stddev_samp"
    )
    sum = sgqlc.types.Field("validation_metrics_sum_order_by", graphql_name="sum")
    var_pop = sgqlc.types.Field("validation_metrics_var_pop_order_by", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("validation_metrics_var_samp_order_by", graphql_name="var_samp")
    variance = sgqlc.types.Field("validation_metrics_variance_order_by", graphql_name="variance")


class validation_metrics_avg_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    raw = sgqlc.types.Field(order_by, graphql_name="raw")
    signed = sgqlc.types.Field(order_by, graphql_name="signed")


class validation_metrics_bool_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_and", "_not", "_or", "id", "raw", "signed")
    _and = sgqlc.types.Field(
        sgqlc.types.list_of("validation_metrics_bool_exp"), graphql_name="_and"
    )
    _not = sgqlc.types.Field("validation_metrics_bool_exp", graphql_name="_not")
    _or = sgqlc.types.Field(sgqlc.types.list_of("validation_metrics_bool_exp"), graphql_name="_or")
    id = sgqlc.types.Field(Int_comparison_exp, graphql_name="id")
    raw = sgqlc.types.Field(float8_comparison_exp, graphql_name="raw")
    signed = sgqlc.types.Field(float8_comparison_exp, graphql_name="signed")


class validation_metrics_max_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    raw = sgqlc.types.Field(order_by, graphql_name="raw")
    signed = sgqlc.types.Field(order_by, graphql_name="signed")


class validation_metrics_min_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    raw = sgqlc.types.Field(order_by, graphql_name="raw")
    signed = sgqlc.types.Field(order_by, graphql_name="signed")


class validation_metrics_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    raw = sgqlc.types.Field(order_by, graphql_name="raw")
    signed = sgqlc.types.Field(order_by, graphql_name="signed")


class validation_metrics_stddev_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    raw = sgqlc.types.Field(order_by, graphql_name="raw")
    signed = sgqlc.types.Field(order_by, graphql_name="signed")


class validation_metrics_stddev_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    raw = sgqlc.types.Field(order_by, graphql_name="raw")
    signed = sgqlc.types.Field(order_by, graphql_name="signed")


class validation_metrics_stddev_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    raw = sgqlc.types.Field(order_by, graphql_name="raw")
    signed = sgqlc.types.Field(order_by, graphql_name="signed")


class validation_metrics_sum_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    raw = sgqlc.types.Field(order_by, graphql_name="raw")
    signed = sgqlc.types.Field(order_by, graphql_name="signed")


class validation_metrics_var_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    raw = sgqlc.types.Field(order_by, graphql_name="raw")
    signed = sgqlc.types.Field(order_by, graphql_name="signed")


class validation_metrics_var_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    raw = sgqlc.types.Field(order_by, graphql_name="raw")
    signed = sgqlc.types.Field(order_by, graphql_name="signed")


class validation_metrics_variance_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    raw = sgqlc.types.Field(order_by, graphql_name="raw")
    signed = sgqlc.types.Field(order_by, graphql_name="signed")


class validation_state_comparison_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("_eq", "_gt", "_gte", "_in", "_is_null", "_lt", "_lte", "_neq", "_nin")
    _eq = sgqlc.types.Field(validation_state, graphql_name="_eq")
    _gt = sgqlc.types.Field(validation_state, graphql_name="_gt")
    _gte = sgqlc.types.Field(validation_state, graphql_name="_gte")
    _in = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(validation_state)), graphql_name="_in"
    )
    _is_null = sgqlc.types.Field(Boolean, graphql_name="_is_null")
    _lt = sgqlc.types.Field(validation_state, graphql_name="_lt")
    _lte = sgqlc.types.Field(validation_state, graphql_name="_lte")
    _neq = sgqlc.types.Field(validation_state, graphql_name="_neq")
    _nin = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null(validation_state)), graphql_name="_nin"
    )


class validations_aggregate_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("validations_avg_order_by", graphql_name="avg")
    count = sgqlc.types.Field(order_by, graphql_name="count")
    max = sgqlc.types.Field("validations_max_order_by", graphql_name="max")
    min = sgqlc.types.Field("validations_min_order_by", graphql_name="min")
    stddev = sgqlc.types.Field("validations_stddev_order_by", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("validations_stddev_pop_order_by", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("validations_stddev_samp_order_by", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("validations_sum_order_by", graphql_name="sum")
    var_pop = sgqlc.types.Field("validations_var_pop_order_by", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("validations_var_samp_order_by", graphql_name="var_samp")
    variance = sgqlc.types.Field("validations_variance_order_by", graphql_name="variance")


class validations_avg_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class validations_bool_exp(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "_and",
        "_not",
        "_or",
        "checkpoint",
        "end_time",
        "id",
        "metric_values",
        "metrics",
        "start_time",
        "state",
        "step",
        "step_id",
        "trial_id",
    )
    _and = sgqlc.types.Field(sgqlc.types.list_of("validations_bool_exp"), graphql_name="_and")
    _not = sgqlc.types.Field("validations_bool_exp", graphql_name="_not")
    _or = sgqlc.types.Field(sgqlc.types.list_of("validations_bool_exp"), graphql_name="_or")
    checkpoint = sgqlc.types.Field(checkpoints_bool_exp, graphql_name="checkpoint")
    end_time = sgqlc.types.Field(timestamptz_comparison_exp, graphql_name="end_time")
    id = sgqlc.types.Field(Int_comparison_exp, graphql_name="id")
    metric_values = sgqlc.types.Field(validation_metrics_bool_exp, graphql_name="metric_values")
    metrics = sgqlc.types.Field(jsonb_comparison_exp, graphql_name="metrics")
    start_time = sgqlc.types.Field(timestamptz_comparison_exp, graphql_name="start_time")
    state = sgqlc.types.Field(validation_state_comparison_exp, graphql_name="state")
    step = sgqlc.types.Field(steps_bool_exp, graphql_name="step")
    step_id = sgqlc.types.Field(Int_comparison_exp, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Int_comparison_exp, graphql_name="trial_id")


class validations_max_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("end_time", "id", "start_time", "step_id", "trial_id")
    end_time = sgqlc.types.Field(order_by, graphql_name="end_time")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    start_time = sgqlc.types.Field(order_by, graphql_name="start_time")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class validations_min_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("end_time", "id", "start_time", "step_id", "trial_id")
    end_time = sgqlc.types.Field(order_by, graphql_name="end_time")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    start_time = sgqlc.types.Field(order_by, graphql_name="start_time")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class validations_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = (
        "checkpoint",
        "end_time",
        "id",
        "metric_values",
        "metrics",
        "start_time",
        "state",
        "step",
        "step_id",
        "trial_id",
    )
    checkpoint = sgqlc.types.Field(checkpoints_order_by, graphql_name="checkpoint")
    end_time = sgqlc.types.Field(order_by, graphql_name="end_time")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    metric_values = sgqlc.types.Field(validation_metrics_order_by, graphql_name="metric_values")
    metrics = sgqlc.types.Field(order_by, graphql_name="metrics")
    start_time = sgqlc.types.Field(order_by, graphql_name="start_time")
    state = sgqlc.types.Field(order_by, graphql_name="state")
    step = sgqlc.types.Field(steps_order_by, graphql_name="step")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class validations_stddev_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class validations_stddev_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class validations_stddev_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class validations_sum_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class validations_var_pop_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class validations_var_samp_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


class validations_variance_order_by(sgqlc.types.Input):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(order_by, graphql_name="id")
    step_id = sgqlc.types.Field(order_by, graphql_name="step_id")
    trial_id = sgqlc.types.Field(order_by, graphql_name="trial_id")


########################################################################
# Output Objects and Interfaces
########################################################################
class agent_user_groups(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("gid", "group_", "id", "uid", "user", "user_", "user_id")
    gid = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="gid")
    group_ = sgqlc.types.Field(sgqlc.types.non_null(String), graphql_name="group_")
    id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="id")
    uid = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="uid")
    user = sgqlc.types.Field(sgqlc.types.non_null("users"), graphql_name="user")
    user_ = sgqlc.types.Field(sgqlc.types.non_null(String), graphql_name="user_")
    user_id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="user_id")


class agent_user_groups_aggregate(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("aggregate", "nodes")
    aggregate = sgqlc.types.Field("agent_user_groups_aggregate_fields", graphql_name="aggregate")
    nodes = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null(agent_user_groups))),
        graphql_name="nodes",
    )


class agent_user_groups_aggregate_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("agent_user_groups_avg_fields", graphql_name="avg")
    count = sgqlc.types.Field(
        Int,
        graphql_name="count",
        args=sgqlc.types.ArgDict(
            (
                (
                    "columns",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(agent_user_groups_select_column)),
                        graphql_name="columns",
                        default=None,
                    ),
                ),
                ("distinct", sgqlc.types.Arg(Boolean, graphql_name="distinct", default=None)),
            )
        ),
    )
    max = sgqlc.types.Field("agent_user_groups_max_fields", graphql_name="max")
    min = sgqlc.types.Field("agent_user_groups_min_fields", graphql_name="min")
    stddev = sgqlc.types.Field("agent_user_groups_stddev_fields", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("agent_user_groups_stddev_pop_fields", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field(
        "agent_user_groups_stddev_samp_fields", graphql_name="stddev_samp"
    )
    sum = sgqlc.types.Field("agent_user_groups_sum_fields", graphql_name="sum")
    var_pop = sgqlc.types.Field("agent_user_groups_var_pop_fields", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("agent_user_groups_var_samp_fields", graphql_name="var_samp")
    variance = sgqlc.types.Field("agent_user_groups_variance_fields", graphql_name="variance")


class agent_user_groups_avg_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(Float, graphql_name="gid")
    id = sgqlc.types.Field(Float, graphql_name="id")
    uid = sgqlc.types.Field(Float, graphql_name="uid")
    user_id = sgqlc.types.Field(Float, graphql_name="user_id")


class agent_user_groups_max_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("gid", "group_", "id", "uid", "user_", "user_id")
    gid = sgqlc.types.Field(Int, graphql_name="gid")
    group_ = sgqlc.types.Field(String, graphql_name="group_")
    id = sgqlc.types.Field(Int, graphql_name="id")
    uid = sgqlc.types.Field(Int, graphql_name="uid")
    user_ = sgqlc.types.Field(String, graphql_name="user_")
    user_id = sgqlc.types.Field(Int, graphql_name="user_id")


class agent_user_groups_min_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("gid", "group_", "id", "uid", "user_", "user_id")
    gid = sgqlc.types.Field(Int, graphql_name="gid")
    group_ = sgqlc.types.Field(String, graphql_name="group_")
    id = sgqlc.types.Field(Int, graphql_name="id")
    uid = sgqlc.types.Field(Int, graphql_name="uid")
    user_ = sgqlc.types.Field(String, graphql_name="user_")
    user_id = sgqlc.types.Field(Int, graphql_name="user_id")


class agent_user_groups_stddev_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(Float, graphql_name="gid")
    id = sgqlc.types.Field(Float, graphql_name="id")
    uid = sgqlc.types.Field(Float, graphql_name="uid")
    user_id = sgqlc.types.Field(Float, graphql_name="user_id")


class agent_user_groups_stddev_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(Float, graphql_name="gid")
    id = sgqlc.types.Field(Float, graphql_name="id")
    uid = sgqlc.types.Field(Float, graphql_name="uid")
    user_id = sgqlc.types.Field(Float, graphql_name="user_id")


class agent_user_groups_stddev_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(Float, graphql_name="gid")
    id = sgqlc.types.Field(Float, graphql_name="id")
    uid = sgqlc.types.Field(Float, graphql_name="uid")
    user_id = sgqlc.types.Field(Float, graphql_name="user_id")


class agent_user_groups_sum_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(Int, graphql_name="gid")
    id = sgqlc.types.Field(Int, graphql_name="id")
    uid = sgqlc.types.Field(Int, graphql_name="uid")
    user_id = sgqlc.types.Field(Int, graphql_name="user_id")


class agent_user_groups_var_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(Float, graphql_name="gid")
    id = sgqlc.types.Field(Float, graphql_name="id")
    uid = sgqlc.types.Field(Float, graphql_name="uid")
    user_id = sgqlc.types.Field(Float, graphql_name="user_id")


class agent_user_groups_var_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(Float, graphql_name="gid")
    id = sgqlc.types.Field(Float, graphql_name="id")
    uid = sgqlc.types.Field(Float, graphql_name="uid")
    user_id = sgqlc.types.Field(Float, graphql_name="user_id")


class agent_user_groups_variance_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("gid", "id", "uid", "user_id")
    gid = sgqlc.types.Field(Float, graphql_name="gid")
    id = sgqlc.types.Field(Float, graphql_name="id")
    uid = sgqlc.types.Field(Float, graphql_name="uid")
    user_id = sgqlc.types.Field(Float, graphql_name="user_id")


class checkpoints(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "end_time",
        "id",
        "labels",
        "resources",
        "start_time",
        "state",
        "step",
        "step_id",
        "trial_id",
        "trials",
        "trials_aggregate",
        "uuid",
        "validation",
    )
    end_time = sgqlc.types.Field(timestamptz, graphql_name="end_time")
    id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="id")
    labels = sgqlc.types.Field(
        jsonb,
        graphql_name="labels",
        args=sgqlc.types.ArgDict(
            (("path", sgqlc.types.Arg(String, graphql_name="path", default=None)),)
        ),
    )
    resources = sgqlc.types.Field(
        jsonb,
        graphql_name="resources",
        args=sgqlc.types.ArgDict(
            (("path", sgqlc.types.Arg(String, graphql_name="path", default=None)),)
        ),
    )
    start_time = sgqlc.types.Field(sgqlc.types.non_null(timestamptz), graphql_name="start_time")
    state = sgqlc.types.Field(sgqlc.types.non_null(checkpoint_state), graphql_name="state")
    step = sgqlc.types.Field("steps", graphql_name="step")
    step_id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="step_id")
    trial_id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="trial_id")
    trials = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("trials"))),
        graphql_name="trials",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(trials_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    trials_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("trials_aggregate"),
        graphql_name="trials_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(trials_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    uuid = sgqlc.types.Field("uuid", graphql_name="uuid")
    validation = sgqlc.types.Field("validations", graphql_name="validation")


class checkpoints_aggregate(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("aggregate", "nodes")
    aggregate = sgqlc.types.Field("checkpoints_aggregate_fields", graphql_name="aggregate")
    nodes = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null(checkpoints))),
        graphql_name="nodes",
    )


class checkpoints_aggregate_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("checkpoints_avg_fields", graphql_name="avg")
    count = sgqlc.types.Field(
        Int,
        graphql_name="count",
        args=sgqlc.types.ArgDict(
            (
                (
                    "columns",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_select_column)),
                        graphql_name="columns",
                        default=None,
                    ),
                ),
                ("distinct", sgqlc.types.Arg(Boolean, graphql_name="distinct", default=None)),
            )
        ),
    )
    max = sgqlc.types.Field("checkpoints_max_fields", graphql_name="max")
    min = sgqlc.types.Field("checkpoints_min_fields", graphql_name="min")
    stddev = sgqlc.types.Field("checkpoints_stddev_fields", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("checkpoints_stddev_pop_fields", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("checkpoints_stddev_samp_fields", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("checkpoints_sum_fields", graphql_name="sum")
    var_pop = sgqlc.types.Field("checkpoints_var_pop_fields", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("checkpoints_var_samp_fields", graphql_name="var_samp")
    variance = sgqlc.types.Field("checkpoints_variance_fields", graphql_name="variance")


class checkpoints_avg_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    step_id = sgqlc.types.Field(Float, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class checkpoints_max_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("end_time", "id", "start_time", "step_id", "trial_id")
    end_time = sgqlc.types.Field(timestamptz, graphql_name="end_time")
    id = sgqlc.types.Field(Int, graphql_name="id")
    start_time = sgqlc.types.Field(timestamptz, graphql_name="start_time")
    step_id = sgqlc.types.Field(Int, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Int, graphql_name="trial_id")


class checkpoints_min_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("end_time", "id", "start_time", "step_id", "trial_id")
    end_time = sgqlc.types.Field(timestamptz, graphql_name="end_time")
    id = sgqlc.types.Field(Int, graphql_name="id")
    start_time = sgqlc.types.Field(timestamptz, graphql_name="start_time")
    step_id = sgqlc.types.Field(Int, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Int, graphql_name="trial_id")


class checkpoints_stddev_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    step_id = sgqlc.types.Field(Float, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class checkpoints_stddev_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    step_id = sgqlc.types.Field(Float, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class checkpoints_stddev_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    step_id = sgqlc.types.Field(Float, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class checkpoints_sum_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Int, graphql_name="id")
    step_id = sgqlc.types.Field(Int, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Int, graphql_name="trial_id")


class checkpoints_var_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    step_id = sgqlc.types.Field(Float, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class checkpoints_var_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    step_id = sgqlc.types.Field(Float, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class checkpoints_variance_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    step_id = sgqlc.types.Field(Float, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class cluster_id(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("cluster_id",)
    cluster_id = sgqlc.types.Field(sgqlc.types.non_null(String), graphql_name="cluster_id")


class cluster_id_aggregate(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("aggregate", "nodes")
    aggregate = sgqlc.types.Field("cluster_id_aggregate_fields", graphql_name="aggregate")
    nodes = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null(cluster_id))),
        graphql_name="nodes",
    )


class cluster_id_aggregate_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("count", "max", "min")
    count = sgqlc.types.Field(
        Int,
        graphql_name="count",
        args=sgqlc.types.ArgDict(
            (
                (
                    "columns",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(cluster_id_select_column)),
                        graphql_name="columns",
                        default=None,
                    ),
                ),
                ("distinct", sgqlc.types.Arg(Boolean, graphql_name="distinct", default=None)),
            )
        ),
    )
    max = sgqlc.types.Field("cluster_id_max_fields", graphql_name="max")
    min = sgqlc.types.Field("cluster_id_min_fields", graphql_name="min")


class cluster_id_max_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("cluster_id",)
    cluster_id = sgqlc.types.Field(String, graphql_name="cluster_id")


class cluster_id_min_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("cluster_id",)
    cluster_id = sgqlc.types.Field(String, graphql_name="cluster_id")


class config_files(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("content", "id")
    content = sgqlc.types.Field(bytea, graphql_name="content")
    id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="id")


class config_files_aggregate(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("aggregate", "nodes")
    aggregate = sgqlc.types.Field("config_files_aggregate_fields", graphql_name="aggregate")
    nodes = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null(config_files))),
        graphql_name="nodes",
    )


class config_files_aggregate_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("config_files_avg_fields", graphql_name="avg")
    count = sgqlc.types.Field(
        Int,
        graphql_name="count",
        args=sgqlc.types.ArgDict(
            (
                (
                    "columns",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(config_files_select_column)),
                        graphql_name="columns",
                        default=None,
                    ),
                ),
                ("distinct", sgqlc.types.Arg(Boolean, graphql_name="distinct", default=None)),
            )
        ),
    )
    max = sgqlc.types.Field("config_files_max_fields", graphql_name="max")
    min = sgqlc.types.Field("config_files_min_fields", graphql_name="min")
    stddev = sgqlc.types.Field("config_files_stddev_fields", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("config_files_stddev_pop_fields", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("config_files_stddev_samp_fields", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("config_files_sum_fields", graphql_name="sum")
    var_pop = sgqlc.types.Field("config_files_var_pop_fields", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("config_files_var_samp_fields", graphql_name="var_samp")
    variance = sgqlc.types.Field("config_files_variance_fields", graphql_name="variance")


class config_files_avg_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Float, graphql_name="id")


class config_files_max_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Int, graphql_name="id")


class config_files_min_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Int, graphql_name="id")


class config_files_stddev_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Float, graphql_name="id")


class config_files_stddev_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Float, graphql_name="id")


class config_files_stddev_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Float, graphql_name="id")


class config_files_sum_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Int, graphql_name="id")


class config_files_var_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Float, graphql_name="id")


class config_files_var_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Float, graphql_name="id")


class config_files_variance_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Float, graphql_name="id")


class experiments(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "archived",
        "best_validation_history",
        "config",
        "end_time",
        "git_commit",
        "git_commit_date",
        "git_committer",
        "git_remote",
        "id",
        "model_definition",
        "model_packages",
        "owner",
        "owner_id",
        "parent_id",
        "progress",
        "searcher_events",
        "searcher_events_aggregate",
        "start_time",
        "state",
        "trials",
        "trials_aggregate",
    )
    archived = sgqlc.types.Field(sgqlc.types.non_null(Boolean), graphql_name="archived")
    best_validation_history = sgqlc.types.Field(
        sgqlc.types.list_of(sgqlc.types.non_null("validations")),
        graphql_name="best_validation_history",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validations_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validations_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(validations_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    config = sgqlc.types.Field(
        sgqlc.types.non_null(jsonb),
        graphql_name="config",
        args=sgqlc.types.ArgDict(
            (("path", sgqlc.types.Arg(String, graphql_name="path", default=None)),)
        ),
    )
    end_time = sgqlc.types.Field(timestamptz, graphql_name="end_time")
    git_commit = sgqlc.types.Field(String, graphql_name="git_commit")
    git_commit_date = sgqlc.types.Field(timestamp, graphql_name="git_commit_date")
    git_committer = sgqlc.types.Field(String, graphql_name="git_committer")
    git_remote = sgqlc.types.Field(String, graphql_name="git_remote")
    id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="id")
    model_definition = sgqlc.types.Field(
        sgqlc.types.non_null(bytea), graphql_name="model_definition"
    )
    model_packages = sgqlc.types.Field(bytea, graphql_name="model_packages")
    owner = sgqlc.types.Field(sgqlc.types.non_null("users"), graphql_name="owner")
    owner_id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="owner_id")
    parent_id = sgqlc.types.Field(Int, graphql_name="parent_id")
    progress = sgqlc.types.Field(float8, graphql_name="progress")
    searcher_events = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("searcher_events"))),
        graphql_name="searcher_events",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(searcher_events_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(searcher_events_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(searcher_events_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    searcher_events_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("searcher_events_aggregate"),
        graphql_name="searcher_events_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(searcher_events_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(searcher_events_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(searcher_events_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    start_time = sgqlc.types.Field(sgqlc.types.non_null(timestamptz), graphql_name="start_time")
    state = sgqlc.types.Field(sgqlc.types.non_null(experiment_state), graphql_name="state")
    trials = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("trials"))),
        graphql_name="trials",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(trials_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    trials_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("trials_aggregate"),
        graphql_name="trials_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(trials_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )


class experiments_aggregate(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("aggregate", "nodes")
    aggregate = sgqlc.types.Field("experiments_aggregate_fields", graphql_name="aggregate")
    nodes = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null(experiments))),
        graphql_name="nodes",
    )


class experiments_aggregate_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("experiments_avg_fields", graphql_name="avg")
    count = sgqlc.types.Field(
        Int,
        graphql_name="count",
        args=sgqlc.types.ArgDict(
            (
                (
                    "columns",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(experiments_select_column)),
                        graphql_name="columns",
                        default=None,
                    ),
                ),
                ("distinct", sgqlc.types.Arg(Boolean, graphql_name="distinct", default=None)),
            )
        ),
    )
    max = sgqlc.types.Field("experiments_max_fields", graphql_name="max")
    min = sgqlc.types.Field("experiments_min_fields", graphql_name="min")
    stddev = sgqlc.types.Field("experiments_stddev_fields", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("experiments_stddev_pop_fields", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("experiments_stddev_samp_fields", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("experiments_sum_fields", graphql_name="sum")
    var_pop = sgqlc.types.Field("experiments_var_pop_fields", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("experiments_var_samp_fields", graphql_name="var_samp")
    variance = sgqlc.types.Field("experiments_variance_fields", graphql_name="variance")


class experiments_avg_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(Float, graphql_name="id")
    owner_id = sgqlc.types.Field(Float, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(Float, graphql_name="parent_id")
    progress = sgqlc.types.Field(Float, graphql_name="progress")


class experiments_max_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "end_time",
        "git_commit",
        "git_committer",
        "git_remote",
        "id",
        "owner_id",
        "parent_id",
        "progress",
        "start_time",
    )
    end_time = sgqlc.types.Field(timestamptz, graphql_name="end_time")
    git_commit = sgqlc.types.Field(String, graphql_name="git_commit")
    git_committer = sgqlc.types.Field(String, graphql_name="git_committer")
    git_remote = sgqlc.types.Field(String, graphql_name="git_remote")
    id = sgqlc.types.Field(Int, graphql_name="id")
    owner_id = sgqlc.types.Field(Int, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(Int, graphql_name="parent_id")
    progress = sgqlc.types.Field(float8, graphql_name="progress")
    start_time = sgqlc.types.Field(timestamptz, graphql_name="start_time")


class experiments_min_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "end_time",
        "git_commit",
        "git_committer",
        "git_remote",
        "id",
        "owner_id",
        "parent_id",
        "progress",
        "start_time",
    )
    end_time = sgqlc.types.Field(timestamptz, graphql_name="end_time")
    git_commit = sgqlc.types.Field(String, graphql_name="git_commit")
    git_committer = sgqlc.types.Field(String, graphql_name="git_committer")
    git_remote = sgqlc.types.Field(String, graphql_name="git_remote")
    id = sgqlc.types.Field(Int, graphql_name="id")
    owner_id = sgqlc.types.Field(Int, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(Int, graphql_name="parent_id")
    progress = sgqlc.types.Field(float8, graphql_name="progress")
    start_time = sgqlc.types.Field(timestamptz, graphql_name="start_time")


class experiments_stddev_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(Float, graphql_name="id")
    owner_id = sgqlc.types.Field(Float, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(Float, graphql_name="parent_id")
    progress = sgqlc.types.Field(Float, graphql_name="progress")


class experiments_stddev_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(Float, graphql_name="id")
    owner_id = sgqlc.types.Field(Float, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(Float, graphql_name="parent_id")
    progress = sgqlc.types.Field(Float, graphql_name="progress")


class experiments_stddev_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(Float, graphql_name="id")
    owner_id = sgqlc.types.Field(Float, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(Float, graphql_name="parent_id")
    progress = sgqlc.types.Field(Float, graphql_name="progress")


class experiments_sum_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(Int, graphql_name="id")
    owner_id = sgqlc.types.Field(Int, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(Int, graphql_name="parent_id")
    progress = sgqlc.types.Field(float8, graphql_name="progress")


class experiments_var_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(Float, graphql_name="id")
    owner_id = sgqlc.types.Field(Float, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(Float, graphql_name="parent_id")
    progress = sgqlc.types.Field(Float, graphql_name="progress")


class experiments_var_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(Float, graphql_name="id")
    owner_id = sgqlc.types.Field(Float, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(Float, graphql_name="parent_id")
    progress = sgqlc.types.Field(Float, graphql_name="progress")


class experiments_variance_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "owner_id", "parent_id", "progress")
    id = sgqlc.types.Field(Float, graphql_name="id")
    owner_id = sgqlc.types.Field(Float, graphql_name="owner_id")
    parent_id = sgqlc.types.Field(Float, graphql_name="parent_id")
    progress = sgqlc.types.Field(Float, graphql_name="progress")


class query_root(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "agent_user_groups",
        "agent_user_groups_aggregate",
        "agent_user_groups_by_pk",
        "best_checkpoint_by_metric",
        "best_checkpoint_by_metric_aggregate",
        "checkpoints",
        "checkpoints_aggregate",
        "checkpoints_by_pk",
        "cluster_id",
        "cluster_id_aggregate",
        "config_files",
        "config_files_aggregate",
        "config_files_by_pk",
        "experiments",
        "experiments_aggregate",
        "experiments_by_pk",
        "searcher_events",
        "searcher_events_aggregate",
        "searcher_events_by_pk",
        "steps",
        "steps_aggregate",
        "steps_by_pk",
        "templates",
        "templates_aggregate",
        "templates_by_pk",
        "trial_logs",
        "trial_logs_aggregate",
        "trial_logs_by_pk",
        "trials",
        "trials_aggregate",
        "trials_by_pk",
        "users",
        "users_aggregate",
        "users_by_pk",
        "validation_metrics",
        "validation_metrics_aggregate",
        "validations",
        "validations_aggregate",
        "validations_by_pk",
    )
    agent_user_groups = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("agent_user_groups"))),
        graphql_name="agent_user_groups",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(agent_user_groups_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(agent_user_groups_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(agent_user_groups_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    agent_user_groups_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("agent_user_groups_aggregate"),
        graphql_name="agent_user_groups_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(agent_user_groups_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(agent_user_groups_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(agent_user_groups_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    agent_user_groups_by_pk = sgqlc.types.Field(
        "agent_user_groups",
        graphql_name="agent_user_groups_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    best_checkpoint_by_metric = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("checkpoints"))),
        graphql_name="best_checkpoint_by_metric",
        args=sgqlc.types.ArgDict(
            (
                (
                    "args",
                    sgqlc.types.Arg(
                        sgqlc.types.non_null(best_checkpoint_by_metric_args),
                        graphql_name="args",
                        default=None,
                    ),
                ),
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(checkpoints_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    best_checkpoint_by_metric_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("checkpoints_aggregate"),
        graphql_name="best_checkpoint_by_metric_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "args",
                    sgqlc.types.Arg(
                        sgqlc.types.non_null(best_checkpoint_by_metric_args),
                        graphql_name="args",
                        default=None,
                    ),
                ),
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(checkpoints_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    checkpoints = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("checkpoints"))),
        graphql_name="checkpoints",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(checkpoints_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    checkpoints_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("checkpoints_aggregate"),
        graphql_name="checkpoints_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(checkpoints_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    checkpoints_by_pk = sgqlc.types.Field(
        "checkpoints",
        graphql_name="checkpoints_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    cluster_id = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("cluster_id"))),
        graphql_name="cluster_id",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(cluster_id_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(cluster_id_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(cluster_id_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    cluster_id_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("cluster_id_aggregate"),
        graphql_name="cluster_id_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(cluster_id_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(cluster_id_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(cluster_id_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    config_files = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("config_files"))),
        graphql_name="config_files",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(config_files_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(config_files_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(config_files_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    config_files_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("config_files_aggregate"),
        graphql_name="config_files_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(config_files_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(config_files_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(config_files_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    config_files_by_pk = sgqlc.types.Field(
        "config_files",
        graphql_name="config_files_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    experiments = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("experiments"))),
        graphql_name="experiments",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(experiments_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(experiments_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(experiments_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    experiments_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("experiments_aggregate"),
        graphql_name="experiments_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(experiments_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(experiments_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(experiments_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    experiments_by_pk = sgqlc.types.Field(
        "experiments",
        graphql_name="experiments_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    searcher_events = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("searcher_events"))),
        graphql_name="searcher_events",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(searcher_events_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(searcher_events_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(searcher_events_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    searcher_events_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("searcher_events_aggregate"),
        graphql_name="searcher_events_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(searcher_events_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(searcher_events_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(searcher_events_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    searcher_events_by_pk = sgqlc.types.Field(
        "searcher_events",
        graphql_name="searcher_events_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    steps = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("steps"))),
        graphql_name="steps",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(steps_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(steps_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(steps_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    steps_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("steps_aggregate"),
        graphql_name="steps_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(steps_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(steps_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(steps_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    steps_by_pk = sgqlc.types.Field(
        "steps",
        graphql_name="steps_by_pk",
        args=sgqlc.types.ArgDict(
            (
                ("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),
                (
                    "trial_id",
                    sgqlc.types.Arg(
                        sgqlc.types.non_null(Int), graphql_name="trial_id", default=None
                    ),
                ),
            )
        ),
    )
    templates = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("templates"))),
        graphql_name="templates",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(templates_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(templates_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(templates_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    templates_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("templates_aggregate"),
        graphql_name="templates_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(templates_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(templates_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(templates_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    templates_by_pk = sgqlc.types.Field(
        "templates",
        graphql_name="templates_by_pk",
        args=sgqlc.types.ArgDict(
            (
                (
                    "name",
                    sgqlc.types.Arg(
                        sgqlc.types.non_null(String), graphql_name="name", default=None
                    ),
                ),
            )
        ),
    )
    trial_logs = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("trial_logs"))),
        graphql_name="trial_logs",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trial_logs_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trial_logs_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(trial_logs_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    trial_logs_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("trial_logs_aggregate"),
        graphql_name="trial_logs_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trial_logs_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trial_logs_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(trial_logs_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    trial_logs_by_pk = sgqlc.types.Field(
        "trial_logs",
        graphql_name="trial_logs_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    trials = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("trials"))),
        graphql_name="trials",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(trials_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    trials_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("trials_aggregate"),
        graphql_name="trials_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(trials_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    trials_by_pk = sgqlc.types.Field(
        "trials",
        graphql_name="trials_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    users = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("users"))),
        graphql_name="users",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(users_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(users_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(users_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    users_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("users_aggregate"),
        graphql_name="users_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(users_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(users_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(users_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    users_by_pk = sgqlc.types.Field(
        "users",
        graphql_name="users_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    validation_metrics = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("validation_metrics"))),
        graphql_name="validation_metrics",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validation_metrics_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validation_metrics_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(
                        validation_metrics_bool_exp, graphql_name="where", default=None
                    ),
                ),
            )
        ),
    )
    validation_metrics_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("validation_metrics_aggregate"),
        graphql_name="validation_metrics_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validation_metrics_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validation_metrics_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(
                        validation_metrics_bool_exp, graphql_name="where", default=None
                    ),
                ),
            )
        ),
    )
    validations = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("validations"))),
        graphql_name="validations",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validations_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validations_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(validations_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    validations_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("validations_aggregate"),
        graphql_name="validations_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validations_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validations_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(validations_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    validations_by_pk = sgqlc.types.Field(
        "validations",
        graphql_name="validations_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )


class searcher_events(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("content", "event_type", "experiment", "experiment_id", "id")
    content = sgqlc.types.Field(
        sgqlc.types.non_null(jsonb),
        graphql_name="content",
        args=sgqlc.types.ArgDict(
            (("path", sgqlc.types.Arg(String, graphql_name="path", default=None)),)
        ),
    )
    event_type = sgqlc.types.Field(sgqlc.types.non_null(String), graphql_name="event_type")
    experiment = sgqlc.types.Field(sgqlc.types.non_null(experiments), graphql_name="experiment")
    experiment_id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="experiment_id")
    id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="id")


class searcher_events_aggregate(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("aggregate", "nodes")
    aggregate = sgqlc.types.Field("searcher_events_aggregate_fields", graphql_name="aggregate")
    nodes = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null(searcher_events))),
        graphql_name="nodes",
    )


class searcher_events_aggregate_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("searcher_events_avg_fields", graphql_name="avg")
    count = sgqlc.types.Field(
        Int,
        graphql_name="count",
        args=sgqlc.types.ArgDict(
            (
                (
                    "columns",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(searcher_events_select_column)),
                        graphql_name="columns",
                        default=None,
                    ),
                ),
                ("distinct", sgqlc.types.Arg(Boolean, graphql_name="distinct", default=None)),
            )
        ),
    )
    max = sgqlc.types.Field("searcher_events_max_fields", graphql_name="max")
    min = sgqlc.types.Field("searcher_events_min_fields", graphql_name="min")
    stddev = sgqlc.types.Field("searcher_events_stddev_fields", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("searcher_events_stddev_pop_fields", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field(
        "searcher_events_stddev_samp_fields", graphql_name="stddev_samp"
    )
    sum = sgqlc.types.Field("searcher_events_sum_fields", graphql_name="sum")
    var_pop = sgqlc.types.Field("searcher_events_var_pop_fields", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("searcher_events_var_samp_fields", graphql_name="var_samp")
    variance = sgqlc.types.Field("searcher_events_variance_fields", graphql_name="variance")


class searcher_events_avg_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(Float, graphql_name="experiment_id")
    id = sgqlc.types.Field(Float, graphql_name="id")


class searcher_events_max_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("event_type", "experiment_id", "id")
    event_type = sgqlc.types.Field(String, graphql_name="event_type")
    experiment_id = sgqlc.types.Field(Int, graphql_name="experiment_id")
    id = sgqlc.types.Field(Int, graphql_name="id")


class searcher_events_min_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("event_type", "experiment_id", "id")
    event_type = sgqlc.types.Field(String, graphql_name="event_type")
    experiment_id = sgqlc.types.Field(Int, graphql_name="experiment_id")
    id = sgqlc.types.Field(Int, graphql_name="id")


class searcher_events_stddev_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(Float, graphql_name="experiment_id")
    id = sgqlc.types.Field(Float, graphql_name="id")


class searcher_events_stddev_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(Float, graphql_name="experiment_id")
    id = sgqlc.types.Field(Float, graphql_name="id")


class searcher_events_stddev_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(Float, graphql_name="experiment_id")
    id = sgqlc.types.Field(Float, graphql_name="id")


class searcher_events_sum_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(Int, graphql_name="experiment_id")
    id = sgqlc.types.Field(Int, graphql_name="id")


class searcher_events_var_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(Float, graphql_name="experiment_id")
    id = sgqlc.types.Field(Float, graphql_name="id")


class searcher_events_var_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(Float, graphql_name="experiment_id")
    id = sgqlc.types.Field(Float, graphql_name="id")


class searcher_events_variance_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id")
    experiment_id = sgqlc.types.Field(Float, graphql_name="experiment_id")
    id = sgqlc.types.Field(Float, graphql_name="id")


class steps(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "checkpoint",
        "end_time",
        "id",
        "metrics",
        "start_time",
        "state",
        "trial",
        "trial_id",
        "validation",
    )
    checkpoint = sgqlc.types.Field(checkpoints, graphql_name="checkpoint")
    end_time = sgqlc.types.Field(timestamptz, graphql_name="end_time")
    id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="id")
    metrics = sgqlc.types.Field(
        jsonb,
        graphql_name="metrics",
        args=sgqlc.types.ArgDict(
            (("path", sgqlc.types.Arg(String, graphql_name="path", default=None)),)
        ),
    )
    start_time = sgqlc.types.Field(sgqlc.types.non_null(timestamptz), graphql_name="start_time")
    state = sgqlc.types.Field(sgqlc.types.non_null(step_state), graphql_name="state")
    trial = sgqlc.types.Field(sgqlc.types.non_null("trials"), graphql_name="trial")
    trial_id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="trial_id")
    validation = sgqlc.types.Field("validations", graphql_name="validation")


class steps_aggregate(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("aggregate", "nodes")
    aggregate = sgqlc.types.Field("steps_aggregate_fields", graphql_name="aggregate")
    nodes = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null(steps))), graphql_name="nodes"
    )


class steps_aggregate_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("steps_avg_fields", graphql_name="avg")
    count = sgqlc.types.Field(
        Int,
        graphql_name="count",
        args=sgqlc.types.ArgDict(
            (
                (
                    "columns",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(steps_select_column)),
                        graphql_name="columns",
                        default=None,
                    ),
                ),
                ("distinct", sgqlc.types.Arg(Boolean, graphql_name="distinct", default=None)),
            )
        ),
    )
    max = sgqlc.types.Field("steps_max_fields", graphql_name="max")
    min = sgqlc.types.Field("steps_min_fields", graphql_name="min")
    stddev = sgqlc.types.Field("steps_stddev_fields", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("steps_stddev_pop_fields", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("steps_stddev_samp_fields", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("steps_sum_fields", graphql_name="sum")
    var_pop = sgqlc.types.Field("steps_var_pop_fields", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("steps_var_samp_fields", graphql_name="var_samp")
    variance = sgqlc.types.Field("steps_variance_fields", graphql_name="variance")


class steps_avg_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class steps_max_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("end_time", "id", "start_time", "trial_id")
    end_time = sgqlc.types.Field(timestamptz, graphql_name="end_time")
    id = sgqlc.types.Field(Int, graphql_name="id")
    start_time = sgqlc.types.Field(timestamptz, graphql_name="start_time")
    trial_id = sgqlc.types.Field(Int, graphql_name="trial_id")


class steps_min_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("end_time", "id", "start_time", "trial_id")
    end_time = sgqlc.types.Field(timestamptz, graphql_name="end_time")
    id = sgqlc.types.Field(Int, graphql_name="id")
    start_time = sgqlc.types.Field(timestamptz, graphql_name="start_time")
    trial_id = sgqlc.types.Field(Int, graphql_name="trial_id")


class steps_stddev_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class steps_stddev_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class steps_stddev_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class steps_sum_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Int, graphql_name="id")
    trial_id = sgqlc.types.Field(Int, graphql_name="trial_id")


class steps_var_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class steps_var_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class steps_variance_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class subscription_root(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "agent_user_groups",
        "agent_user_groups_aggregate",
        "agent_user_groups_by_pk",
        "best_checkpoint_by_metric",
        "best_checkpoint_by_metric_aggregate",
        "checkpoints",
        "checkpoints_aggregate",
        "checkpoints_by_pk",
        "cluster_id",
        "cluster_id_aggregate",
        "config_files",
        "config_files_aggregate",
        "config_files_by_pk",
        "experiments",
        "experiments_aggregate",
        "experiments_by_pk",
        "searcher_events",
        "searcher_events_aggregate",
        "searcher_events_by_pk",
        "steps",
        "steps_aggregate",
        "steps_by_pk",
        "templates",
        "templates_aggregate",
        "templates_by_pk",
        "trial_logs",
        "trial_logs_aggregate",
        "trial_logs_by_pk",
        "trials",
        "trials_aggregate",
        "trials_by_pk",
        "users",
        "users_aggregate",
        "users_by_pk",
        "validation_metrics",
        "validation_metrics_aggregate",
        "validations",
        "validations_aggregate",
        "validations_by_pk",
    )
    agent_user_groups = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("agent_user_groups"))),
        graphql_name="agent_user_groups",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(agent_user_groups_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(agent_user_groups_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(agent_user_groups_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    agent_user_groups_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("agent_user_groups_aggregate"),
        graphql_name="agent_user_groups_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(agent_user_groups_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(agent_user_groups_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(agent_user_groups_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    agent_user_groups_by_pk = sgqlc.types.Field(
        "agent_user_groups",
        graphql_name="agent_user_groups_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    best_checkpoint_by_metric = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("checkpoints"))),
        graphql_name="best_checkpoint_by_metric",
        args=sgqlc.types.ArgDict(
            (
                (
                    "args",
                    sgqlc.types.Arg(
                        sgqlc.types.non_null(best_checkpoint_by_metric_args),
                        graphql_name="args",
                        default=None,
                    ),
                ),
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(checkpoints_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    best_checkpoint_by_metric_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("checkpoints_aggregate"),
        graphql_name="best_checkpoint_by_metric_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "args",
                    sgqlc.types.Arg(
                        sgqlc.types.non_null(best_checkpoint_by_metric_args),
                        graphql_name="args",
                        default=None,
                    ),
                ),
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(checkpoints_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    checkpoints = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("checkpoints"))),
        graphql_name="checkpoints",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(checkpoints_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    checkpoints_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("checkpoints_aggregate"),
        graphql_name="checkpoints_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(checkpoints_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    checkpoints_by_pk = sgqlc.types.Field(
        "checkpoints",
        graphql_name="checkpoints_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    cluster_id = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("cluster_id"))),
        graphql_name="cluster_id",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(cluster_id_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(cluster_id_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(cluster_id_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    cluster_id_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("cluster_id_aggregate"),
        graphql_name="cluster_id_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(cluster_id_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(cluster_id_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(cluster_id_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    config_files = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("config_files"))),
        graphql_name="config_files",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(config_files_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(config_files_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(config_files_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    config_files_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("config_files_aggregate"),
        graphql_name="config_files_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(config_files_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(config_files_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(config_files_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    config_files_by_pk = sgqlc.types.Field(
        "config_files",
        graphql_name="config_files_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    experiments = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("experiments"))),
        graphql_name="experiments",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(experiments_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(experiments_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(experiments_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    experiments_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("experiments_aggregate"),
        graphql_name="experiments_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(experiments_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(experiments_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(experiments_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    experiments_by_pk = sgqlc.types.Field(
        "experiments",
        graphql_name="experiments_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    searcher_events = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("searcher_events"))),
        graphql_name="searcher_events",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(searcher_events_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(searcher_events_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(searcher_events_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    searcher_events_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("searcher_events_aggregate"),
        graphql_name="searcher_events_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(searcher_events_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(searcher_events_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(searcher_events_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    searcher_events_by_pk = sgqlc.types.Field(
        "searcher_events",
        graphql_name="searcher_events_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    steps = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("steps"))),
        graphql_name="steps",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(steps_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(steps_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(steps_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    steps_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("steps_aggregate"),
        graphql_name="steps_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(steps_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(steps_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(steps_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    steps_by_pk = sgqlc.types.Field(
        "steps",
        graphql_name="steps_by_pk",
        args=sgqlc.types.ArgDict(
            (
                ("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),
                (
                    "trial_id",
                    sgqlc.types.Arg(
                        sgqlc.types.non_null(Int), graphql_name="trial_id", default=None
                    ),
                ),
            )
        ),
    )
    templates = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("templates"))),
        graphql_name="templates",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(templates_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(templates_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(templates_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    templates_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("templates_aggregate"),
        graphql_name="templates_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(templates_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(templates_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(templates_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    templates_by_pk = sgqlc.types.Field(
        "templates",
        graphql_name="templates_by_pk",
        args=sgqlc.types.ArgDict(
            (
                (
                    "name",
                    sgqlc.types.Arg(
                        sgqlc.types.non_null(String), graphql_name="name", default=None
                    ),
                ),
            )
        ),
    )
    trial_logs = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("trial_logs"))),
        graphql_name="trial_logs",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trial_logs_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trial_logs_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(trial_logs_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    trial_logs_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("trial_logs_aggregate"),
        graphql_name="trial_logs_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trial_logs_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trial_logs_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(trial_logs_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    trial_logs_by_pk = sgqlc.types.Field(
        "trial_logs",
        graphql_name="trial_logs_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    trials = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("trials"))),
        graphql_name="trials",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(trials_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    trials_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("trials_aggregate"),
        graphql_name="trials_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(trials_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    trials_by_pk = sgqlc.types.Field(
        "trials",
        graphql_name="trials_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    users = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("users"))),
        graphql_name="users",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(users_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(users_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(users_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    users_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("users_aggregate"),
        graphql_name="users_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(users_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(users_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(users_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    users_by_pk = sgqlc.types.Field(
        "users",
        graphql_name="users_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )
    validation_metrics = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("validation_metrics"))),
        graphql_name="validation_metrics",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validation_metrics_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validation_metrics_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(
                        validation_metrics_bool_exp, graphql_name="where", default=None
                    ),
                ),
            )
        ),
    )
    validation_metrics_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("validation_metrics_aggregate"),
        graphql_name="validation_metrics_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validation_metrics_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validation_metrics_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(
                        validation_metrics_bool_exp, graphql_name="where", default=None
                    ),
                ),
            )
        ),
    )
    validations = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("validations"))),
        graphql_name="validations",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validations_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validations_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(validations_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    validations_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("validations_aggregate"),
        graphql_name="validations_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validations_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validations_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(validations_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    validations_by_pk = sgqlc.types.Field(
        "validations",
        graphql_name="validations_by_pk",
        args=sgqlc.types.ArgDict(
            (("id", sgqlc.types.Arg(sgqlc.types.non_null(Int), graphql_name="id", default=None)),)
        ),
    )


class templates(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("config", "name")
    config = sgqlc.types.Field(
        sgqlc.types.non_null(jsonb),
        graphql_name="config",
        args=sgqlc.types.ArgDict(
            (("path", sgqlc.types.Arg(String, graphql_name="path", default=None)),)
        ),
    )
    name = sgqlc.types.Field(sgqlc.types.non_null(String), graphql_name="name")


class templates_aggregate(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("aggregate", "nodes")
    aggregate = sgqlc.types.Field("templates_aggregate_fields", graphql_name="aggregate")
    nodes = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null(templates))),
        graphql_name="nodes",
    )


class templates_aggregate_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("count", "max", "min")
    count = sgqlc.types.Field(
        Int,
        graphql_name="count",
        args=sgqlc.types.ArgDict(
            (
                (
                    "columns",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(templates_select_column)),
                        graphql_name="columns",
                        default=None,
                    ),
                ),
                ("distinct", sgqlc.types.Arg(Boolean, graphql_name="distinct", default=None)),
            )
        ),
    )
    max = sgqlc.types.Field("templates_max_fields", graphql_name="max")
    min = sgqlc.types.Field("templates_min_fields", graphql_name="min")


class templates_max_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("name",)
    name = sgqlc.types.Field(String, graphql_name="name")


class templates_min_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("name",)
    name = sgqlc.types.Field(String, graphql_name="name")


class trial_logs(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "message", "trial", "trial_id")
    id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="id")
    message = sgqlc.types.Field(sgqlc.types.non_null(bytea), graphql_name="message")
    trial = sgqlc.types.Field(sgqlc.types.non_null("trials"), graphql_name="trial")
    trial_id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="trial_id")


class trial_logs_aggregate(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("aggregate", "nodes")
    aggregate = sgqlc.types.Field("trial_logs_aggregate_fields", graphql_name="aggregate")
    nodes = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null(trial_logs))),
        graphql_name="nodes",
    )


class trial_logs_aggregate_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("trial_logs_avg_fields", graphql_name="avg")
    count = sgqlc.types.Field(
        Int,
        graphql_name="count",
        args=sgqlc.types.ArgDict(
            (
                (
                    "columns",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trial_logs_select_column)),
                        graphql_name="columns",
                        default=None,
                    ),
                ),
                ("distinct", sgqlc.types.Arg(Boolean, graphql_name="distinct", default=None)),
            )
        ),
    )
    max = sgqlc.types.Field("trial_logs_max_fields", graphql_name="max")
    min = sgqlc.types.Field("trial_logs_min_fields", graphql_name="min")
    stddev = sgqlc.types.Field("trial_logs_stddev_fields", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("trial_logs_stddev_pop_fields", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("trial_logs_stddev_samp_fields", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("trial_logs_sum_fields", graphql_name="sum")
    var_pop = sgqlc.types.Field("trial_logs_var_pop_fields", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("trial_logs_var_samp_fields", graphql_name="var_samp")
    variance = sgqlc.types.Field("trial_logs_variance_fields", graphql_name="variance")


class trial_logs_avg_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class trial_logs_max_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Int, graphql_name="id")
    trial_id = sgqlc.types.Field(Int, graphql_name="trial_id")


class trial_logs_min_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Int, graphql_name="id")
    trial_id = sgqlc.types.Field(Int, graphql_name="trial_id")


class trial_logs_stddev_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class trial_logs_stddev_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class trial_logs_stddev_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class trial_logs_sum_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Int, graphql_name="id")
    trial_id = sgqlc.types.Field(Int, graphql_name="trial_id")


class trial_logs_var_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class trial_logs_var_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class trial_logs_variance_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class trials(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "checkpoint",
        "checkpoints",
        "checkpoints_aggregate",
        "end_time",
        "experiment",
        "experiment_id",
        "hparams",
        "id",
        "seed",
        "start_time",
        "state",
        "steps",
        "steps_aggregate",
        "trial_logs",
        "trial_logs_aggregate",
        "validations",
        "validations_aggregate",
        "warm_start_checkpoint_id",
    )
    checkpoint = sgqlc.types.Field("checkpoints", graphql_name="checkpoint")
    checkpoints = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("checkpoints"))),
        graphql_name="checkpoints",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(checkpoints_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    checkpoints_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("checkpoints_aggregate"),
        graphql_name="checkpoints_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(checkpoints_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(checkpoints_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    end_time = sgqlc.types.Field(timestamptz, graphql_name="end_time")
    experiment = sgqlc.types.Field(sgqlc.types.non_null(experiments), graphql_name="experiment")
    experiment_id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="experiment_id")
    hparams = sgqlc.types.Field(
        sgqlc.types.non_null(jsonb),
        graphql_name="hparams",
        args=sgqlc.types.ArgDict(
            (("path", sgqlc.types.Arg(String, graphql_name="path", default=None)),)
        ),
    )
    id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="id")
    seed = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="seed")
    start_time = sgqlc.types.Field(sgqlc.types.non_null(timestamptz), graphql_name="start_time")
    state = sgqlc.types.Field(sgqlc.types.non_null(trial_state), graphql_name="state")
    steps = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("steps"))),
        graphql_name="steps",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(steps_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(steps_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(steps_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    steps_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("steps_aggregate"),
        graphql_name="steps_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(steps_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(steps_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(steps_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    trial_logs = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("trial_logs"))),
        graphql_name="trial_logs",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trial_logs_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trial_logs_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(trial_logs_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    trial_logs_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("trial_logs_aggregate"),
        graphql_name="trial_logs_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trial_logs_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trial_logs_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                ("where", sgqlc.types.Arg(trial_logs_bool_exp, graphql_name="where", default=None)),
            )
        ),
    )
    validations = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("validations"))),
        graphql_name="validations",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validations_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validations_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(validations_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    validations_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("validations_aggregate"),
        graphql_name="validations_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validations_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validations_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(validations_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    warm_start_checkpoint_id = sgqlc.types.Field(Int, graphql_name="warm_start_checkpoint_id")


class trials_aggregate(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("aggregate", "nodes")
    aggregate = sgqlc.types.Field("trials_aggregate_fields", graphql_name="aggregate")
    nodes = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null(trials))),
        graphql_name="nodes",
    )


class trials_aggregate_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("trials_avg_fields", graphql_name="avg")
    count = sgqlc.types.Field(
        Int,
        graphql_name="count",
        args=sgqlc.types.ArgDict(
            (
                (
                    "columns",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(trials_select_column)),
                        graphql_name="columns",
                        default=None,
                    ),
                ),
                ("distinct", sgqlc.types.Arg(Boolean, graphql_name="distinct", default=None)),
            )
        ),
    )
    max = sgqlc.types.Field("trials_max_fields", graphql_name="max")
    min = sgqlc.types.Field("trials_min_fields", graphql_name="min")
    stddev = sgqlc.types.Field("trials_stddev_fields", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("trials_stddev_pop_fields", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("trials_stddev_samp_fields", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("trials_sum_fields", graphql_name="sum")
    var_pop = sgqlc.types.Field("trials_var_pop_fields", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("trials_var_samp_fields", graphql_name="var_samp")
    variance = sgqlc.types.Field("trials_variance_fields", graphql_name="variance")


class trials_avg_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(Float, graphql_name="experiment_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    seed = sgqlc.types.Field(Float, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(Float, graphql_name="warm_start_checkpoint_id")


class trials_max_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "end_time",
        "experiment_id",
        "id",
        "seed",
        "start_time",
        "warm_start_checkpoint_id",
    )
    end_time = sgqlc.types.Field(timestamptz, graphql_name="end_time")
    experiment_id = sgqlc.types.Field(Int, graphql_name="experiment_id")
    id = sgqlc.types.Field(Int, graphql_name="id")
    seed = sgqlc.types.Field(Int, graphql_name="seed")
    start_time = sgqlc.types.Field(timestamptz, graphql_name="start_time")
    warm_start_checkpoint_id = sgqlc.types.Field(Int, graphql_name="warm_start_checkpoint_id")


class trials_min_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "end_time",
        "experiment_id",
        "id",
        "seed",
        "start_time",
        "warm_start_checkpoint_id",
    )
    end_time = sgqlc.types.Field(timestamptz, graphql_name="end_time")
    experiment_id = sgqlc.types.Field(Int, graphql_name="experiment_id")
    id = sgqlc.types.Field(Int, graphql_name="id")
    seed = sgqlc.types.Field(Int, graphql_name="seed")
    start_time = sgqlc.types.Field(timestamptz, graphql_name="start_time")
    warm_start_checkpoint_id = sgqlc.types.Field(Int, graphql_name="warm_start_checkpoint_id")


class trials_stddev_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(Float, graphql_name="experiment_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    seed = sgqlc.types.Field(Float, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(Float, graphql_name="warm_start_checkpoint_id")


class trials_stddev_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(Float, graphql_name="experiment_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    seed = sgqlc.types.Field(Float, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(Float, graphql_name="warm_start_checkpoint_id")


class trials_stddev_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(Float, graphql_name="experiment_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    seed = sgqlc.types.Field(Float, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(Float, graphql_name="warm_start_checkpoint_id")


class trials_sum_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(Int, graphql_name="experiment_id")
    id = sgqlc.types.Field(Int, graphql_name="id")
    seed = sgqlc.types.Field(Int, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(Int, graphql_name="warm_start_checkpoint_id")


class trials_var_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(Float, graphql_name="experiment_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    seed = sgqlc.types.Field(Float, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(Float, graphql_name="warm_start_checkpoint_id")


class trials_var_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(Float, graphql_name="experiment_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    seed = sgqlc.types.Field(Float, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(Float, graphql_name="warm_start_checkpoint_id")


class trials_variance_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("experiment_id", "id", "seed", "warm_start_checkpoint_id")
    experiment_id = sgqlc.types.Field(Float, graphql_name="experiment_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    seed = sgqlc.types.Field(Float, graphql_name="seed")
    warm_start_checkpoint_id = sgqlc.types.Field(Float, graphql_name="warm_start_checkpoint_id")


class users(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "active",
        "admin",
        "agent_user_group",
        "experiments",
        "experiments_aggregate",
        "id",
        "username",
    )
    active = sgqlc.types.Field(Boolean, graphql_name="active")
    admin = sgqlc.types.Field(Boolean, graphql_name="admin")
    agent_user_group = sgqlc.types.Field(agent_user_groups, graphql_name="agent_user_group")
    experiments = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null("experiments"))),
        graphql_name="experiments",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(experiments_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(experiments_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(experiments_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    experiments_aggregate = sgqlc.types.Field(
        sgqlc.types.non_null("experiments_aggregate"),
        graphql_name="experiments_aggregate",
        args=sgqlc.types.ArgDict(
            (
                (
                    "distinct_on",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(experiments_select_column)),
                        graphql_name="distinct_on",
                        default=None,
                    ),
                ),
                ("limit", sgqlc.types.Arg(Int, graphql_name="limit", default=None)),
                ("offset", sgqlc.types.Arg(Int, graphql_name="offset", default=None)),
                (
                    "order_by",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(experiments_order_by)),
                        graphql_name="order_by",
                        default=None,
                    ),
                ),
                (
                    "where",
                    sgqlc.types.Arg(experiments_bool_exp, graphql_name="where", default=None),
                ),
            )
        ),
    )
    id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="id")
    username = sgqlc.types.Field(sgqlc.types.non_null(String), graphql_name="username")


class users_aggregate(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("aggregate", "nodes")
    aggregate = sgqlc.types.Field("users_aggregate_fields", graphql_name="aggregate")
    nodes = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null(users))), graphql_name="nodes"
    )


class users_aggregate_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("users_avg_fields", graphql_name="avg")
    count = sgqlc.types.Field(
        Int,
        graphql_name="count",
        args=sgqlc.types.ArgDict(
            (
                (
                    "columns",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(users_select_column)),
                        graphql_name="columns",
                        default=None,
                    ),
                ),
                ("distinct", sgqlc.types.Arg(Boolean, graphql_name="distinct", default=None)),
            )
        ),
    )
    max = sgqlc.types.Field("users_max_fields", graphql_name="max")
    min = sgqlc.types.Field("users_min_fields", graphql_name="min")
    stddev = sgqlc.types.Field("users_stddev_fields", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("users_stddev_pop_fields", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("users_stddev_samp_fields", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("users_sum_fields", graphql_name="sum")
    var_pop = sgqlc.types.Field("users_var_pop_fields", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("users_var_samp_fields", graphql_name="var_samp")
    variance = sgqlc.types.Field("users_variance_fields", graphql_name="variance")


class users_avg_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Float, graphql_name="id")


class users_max_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "username")
    id = sgqlc.types.Field(Int, graphql_name="id")
    username = sgqlc.types.Field(String, graphql_name="username")


class users_min_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "username")
    id = sgqlc.types.Field(Int, graphql_name="id")
    username = sgqlc.types.Field(String, graphql_name="username")


class users_stddev_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Float, graphql_name="id")


class users_stddev_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Float, graphql_name="id")


class users_stddev_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Float, graphql_name="id")


class users_sum_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Int, graphql_name="id")


class users_var_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Float, graphql_name="id")


class users_var_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Float, graphql_name="id")


class users_variance_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id",)
    id = sgqlc.types.Field(Float, graphql_name="id")


class validation_metrics(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(Int, graphql_name="id")
    raw = sgqlc.types.Field(float8, graphql_name="raw")
    signed = sgqlc.types.Field(float8, graphql_name="signed")


class validation_metrics_aggregate(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("aggregate", "nodes")
    aggregate = sgqlc.types.Field("validation_metrics_aggregate_fields", graphql_name="aggregate")
    nodes = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null(validation_metrics))),
        graphql_name="nodes",
    )


class validation_metrics_aggregate_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("validation_metrics_avg_fields", graphql_name="avg")
    count = sgqlc.types.Field(
        Int,
        graphql_name="count",
        args=sgqlc.types.ArgDict(
            (
                (
                    "columns",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validation_metrics_select_column)),
                        graphql_name="columns",
                        default=None,
                    ),
                ),
                ("distinct", sgqlc.types.Arg(Boolean, graphql_name="distinct", default=None)),
            )
        ),
    )
    max = sgqlc.types.Field("validation_metrics_max_fields", graphql_name="max")
    min = sgqlc.types.Field("validation_metrics_min_fields", graphql_name="min")
    stddev = sgqlc.types.Field("validation_metrics_stddev_fields", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field(
        "validation_metrics_stddev_pop_fields", graphql_name="stddev_pop"
    )
    stddev_samp = sgqlc.types.Field(
        "validation_metrics_stddev_samp_fields", graphql_name="stddev_samp"
    )
    sum = sgqlc.types.Field("validation_metrics_sum_fields", graphql_name="sum")
    var_pop = sgqlc.types.Field("validation_metrics_var_pop_fields", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("validation_metrics_var_samp_fields", graphql_name="var_samp")
    variance = sgqlc.types.Field("validation_metrics_variance_fields", graphql_name="variance")


class validation_metrics_avg_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(Float, graphql_name="id")
    raw = sgqlc.types.Field(Float, graphql_name="raw")
    signed = sgqlc.types.Field(Float, graphql_name="signed")


class validation_metrics_max_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(Int, graphql_name="id")
    raw = sgqlc.types.Field(float8, graphql_name="raw")
    signed = sgqlc.types.Field(float8, graphql_name="signed")


class validation_metrics_min_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(Int, graphql_name="id")
    raw = sgqlc.types.Field(float8, graphql_name="raw")
    signed = sgqlc.types.Field(float8, graphql_name="signed")


class validation_metrics_stddev_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(Float, graphql_name="id")
    raw = sgqlc.types.Field(Float, graphql_name="raw")
    signed = sgqlc.types.Field(Float, graphql_name="signed")


class validation_metrics_stddev_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(Float, graphql_name="id")
    raw = sgqlc.types.Field(Float, graphql_name="raw")
    signed = sgqlc.types.Field(Float, graphql_name="signed")


class validation_metrics_stddev_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(Float, graphql_name="id")
    raw = sgqlc.types.Field(Float, graphql_name="raw")
    signed = sgqlc.types.Field(Float, graphql_name="signed")


class validation_metrics_sum_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(Int, graphql_name="id")
    raw = sgqlc.types.Field(float8, graphql_name="raw")
    signed = sgqlc.types.Field(float8, graphql_name="signed")


class validation_metrics_var_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(Float, graphql_name="id")
    raw = sgqlc.types.Field(Float, graphql_name="raw")
    signed = sgqlc.types.Field(Float, graphql_name="signed")


class validation_metrics_var_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(Float, graphql_name="id")
    raw = sgqlc.types.Field(Float, graphql_name="raw")
    signed = sgqlc.types.Field(Float, graphql_name="signed")


class validation_metrics_variance_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "raw", "signed")
    id = sgqlc.types.Field(Float, graphql_name="id")
    raw = sgqlc.types.Field(Float, graphql_name="raw")
    signed = sgqlc.types.Field(Float, graphql_name="signed")


class validations(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "checkpoint",
        "end_time",
        "id",
        "metric_values",
        "metrics",
        "start_time",
        "state",
        "step",
        "step_id",
        "trial_id",
    )
    checkpoint = sgqlc.types.Field(checkpoints, graphql_name="checkpoint")
    end_time = sgqlc.types.Field(timestamptz, graphql_name="end_time")
    id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="id")
    metric_values = sgqlc.types.Field(validation_metrics, graphql_name="metric_values")
    metrics = sgqlc.types.Field(
        jsonb,
        graphql_name="metrics",
        args=sgqlc.types.ArgDict(
            (("path", sgqlc.types.Arg(String, graphql_name="path", default=None)),)
        ),
    )
    start_time = sgqlc.types.Field(sgqlc.types.non_null(timestamptz), graphql_name="start_time")
    state = sgqlc.types.Field(sgqlc.types.non_null(validation_state), graphql_name="state")
    step = sgqlc.types.Field(steps, graphql_name="step")
    step_id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="step_id")
    trial_id = sgqlc.types.Field(sgqlc.types.non_null(Int), graphql_name="trial_id")


class validations_aggregate(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("aggregate", "nodes")
    aggregate = sgqlc.types.Field("validations_aggregate_fields", graphql_name="aggregate")
    nodes = sgqlc.types.Field(
        sgqlc.types.non_null(sgqlc.types.list_of(sgqlc.types.non_null(validations))),
        graphql_name="nodes",
    )


class validations_aggregate_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = (
        "avg",
        "count",
        "max",
        "min",
        "stddev",
        "stddev_pop",
        "stddev_samp",
        "sum",
        "var_pop",
        "var_samp",
        "variance",
    )
    avg = sgqlc.types.Field("validations_avg_fields", graphql_name="avg")
    count = sgqlc.types.Field(
        Int,
        graphql_name="count",
        args=sgqlc.types.ArgDict(
            (
                (
                    "columns",
                    sgqlc.types.Arg(
                        sgqlc.types.list_of(sgqlc.types.non_null(validations_select_column)),
                        graphql_name="columns",
                        default=None,
                    ),
                ),
                ("distinct", sgqlc.types.Arg(Boolean, graphql_name="distinct", default=None)),
            )
        ),
    )
    max = sgqlc.types.Field("validations_max_fields", graphql_name="max")
    min = sgqlc.types.Field("validations_min_fields", graphql_name="min")
    stddev = sgqlc.types.Field("validations_stddev_fields", graphql_name="stddev")
    stddev_pop = sgqlc.types.Field("validations_stddev_pop_fields", graphql_name="stddev_pop")
    stddev_samp = sgqlc.types.Field("validations_stddev_samp_fields", graphql_name="stddev_samp")
    sum = sgqlc.types.Field("validations_sum_fields", graphql_name="sum")
    var_pop = sgqlc.types.Field("validations_var_pop_fields", graphql_name="var_pop")
    var_samp = sgqlc.types.Field("validations_var_samp_fields", graphql_name="var_samp")
    variance = sgqlc.types.Field("validations_variance_fields", graphql_name="variance")


class validations_avg_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    step_id = sgqlc.types.Field(Float, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class validations_max_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("end_time", "id", "start_time", "step_id", "trial_id")
    end_time = sgqlc.types.Field(timestamptz, graphql_name="end_time")
    id = sgqlc.types.Field(Int, graphql_name="id")
    start_time = sgqlc.types.Field(timestamptz, graphql_name="start_time")
    step_id = sgqlc.types.Field(Int, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Int, graphql_name="trial_id")


class validations_min_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("end_time", "id", "start_time", "step_id", "trial_id")
    end_time = sgqlc.types.Field(timestamptz, graphql_name="end_time")
    id = sgqlc.types.Field(Int, graphql_name="id")
    start_time = sgqlc.types.Field(timestamptz, graphql_name="start_time")
    step_id = sgqlc.types.Field(Int, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Int, graphql_name="trial_id")


class validations_stddev_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    step_id = sgqlc.types.Field(Float, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class validations_stddev_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    step_id = sgqlc.types.Field(Float, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class validations_stddev_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    step_id = sgqlc.types.Field(Float, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class validations_sum_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Int, graphql_name="id")
    step_id = sgqlc.types.Field(Int, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Int, graphql_name="trial_id")


class validations_var_pop_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    step_id = sgqlc.types.Field(Float, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class validations_var_samp_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    step_id = sgqlc.types.Field(Float, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


class validations_variance_fields(sgqlc.types.Type):
    __schema__ = gql
    __field_names__ = ("id", "step_id", "trial_id")
    id = sgqlc.types.Field(Float, graphql_name="id")
    step_id = sgqlc.types.Field(Float, graphql_name="step_id")
    trial_id = sgqlc.types.Field(Float, graphql_name="trial_id")


########################################################################
# Unions
########################################################################

########################################################################
# Schema Entry Points
########################################################################
gql.query_type = query_root
gql.mutation_type = None
gql.subscription_type = subscription_root
