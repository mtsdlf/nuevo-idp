{
  "architecture": {
    "style": "workflow-centric-control-plane",
    "core_principles": [
      "intent_separated_from_execution",
      "domain_as_source_of_truth",
      "durable_state_over_ephemeral_jobs",
      "auditability_first",
      "provider_agnosticism"
    ]
  },

  "service_boundaries": {
    "control_plane_api": {
      "responsibility": [
        "receive_commands",
        "validate_domain_rules",
        "mutate_domain_state",
        "emit_domain_events",
        "serve_queries"
      ],
      "must_not": [
        "call_external_providers",
        "execute_side_effects",
        "contain_retry_logic"
      ]
    },
    "workflow_engine": {
      "responsibility": [
        "orchestrate_long_running_processes",
        "persist_workflow_state",
        "handle_retries_and_timeouts",
        "support_pause_and_resume"
      ],
      "patterns": ["saga_orchestration", "deterministic_execution"]
    },
    "execution_workers": {
      "responsibility": [
        "execute_side_effects",
        "integrate_external_systems",
        "ensure_idempotency",
        "normalize_external_errors"
      ],
      "must_not": [
        "decide_business_state",
        "mutate_domain_directly",
        "accept_human_requests"
      ]
    }
  },

  "architectural_patterns": {
    "hexagonal_architecture": {
      "applies_to": ["api", "workers"],
      "structure": ["domain", "application", "ports", "adapters"]
    },
    "orchestration_over_choreography": {
      "where": "workflow_engine",
      "reason": ["explicit_ordering", "auditability", "pause_resume"]
    },
    "cqrs_pragmatic": {
      "commands": "mutate_state",
      "queries": "read_models"
    }
  },

  "implementation_structure": {
    "language": "go",
    "directories": {
      "cmd": ["api", "worker"],
      "internal": [
        "domain",
        "application",
        "ports",
        "workflow",
        "adapters",
        "security"
      ]
    },
    "shared_platform": {
      "module": "github.com/nuevo-idp/platform",
      "packages": {
        "observability": [
          "NewLogger",
          "LoggerWithTrace",
          "InitMetrics",
          "InstrumentHTTP",
          "ObserveDomainEvent"
        ],
        "httpx": [
          "WriteJSON",
          "WriteText",
          "RequireMethod",
          "DecodeJSON"
        ],
        "config": [
          "Get",
          "Require"
        ],
        "errors": [
          "Kind",
          "Error",
          "Domain",
          "Validation",
          "Conflict",
          "NotFound",
          "Internal",
          "IsKind",
          "Code"
        ]
      },
      "rules": [
        "services_no_definen_observability_local",
        "handlers_http_usan_platform_httpx_no_helpers_ad_hoc",
        "lectura_de_envs_via_platform_config_no_os_getenv_en_servicios",
        "errores_de_dominio_usar_platform_errors_con_mapeo_a_http_por_kind"
      ]
    }
  },

  "observability": {
    "tracing": {
      "standard": "opentelemetry",
      "rules": [
        "one_trace_per_request_or_workflow",
        "one_span_per_workflow_step",
        "child_spans_for_external_calls"
      ]
    },
    "metrics": {
      "focus": [
        "workflow_duration",
        "retry_count",
        "blocked_workflows",
        "provider_error_rate"
      ],
      "constraints": ["low_cardinality_only"],
      "domain_events": {
        "metric_name": "domain_events_total",
        "labels": {
          "event": "nombre_estable_de_caso_de_uso",
          "result": "resultado_binarizado_success_o_error"
        },
        "naming_convention": {
          "http_commands": "<aggregate>_<action>",
          "workflows": "workflow_<aggregate>_<flow>",
          "workers": "<integration>_<action>"
        },
        "examples": [
          "application_created",
          "application_approved",
          "application_environment_provisioning_completed",
          "secret_rotation_started",
          "secret_rotation_completed",
          "workflow_application_onboarding_completed",
          "workflow_appenv_provisioning_failed",
          "workflow_secret_rotation_completed",
          "github_repo_created",
          "appenv_side_effect_accepted"
        ]
      }
    },
    "logging": {
      "library": "zap",
      "format": "json",
      "rules": [
        "logs_explain_execution_not_business",
        "trace_id_required",
        "no_secrets_in_logs"
      ]
    }
  },

  "testing_practices": {
    "domain": {
      "approach": "tdd",
      "rules": ["no_io", "no_mocks"]
    },
    "workflows": {
      "approach": "integration",
      "rules": ["real_engine_test_env", "fake_adapters"]
    },
    "adapters": {
      "approach": "contract_testing"
    }
  },

  "linting_and_code_quality": {
    "tool": "golangci-lint",
    "linters_core": [
      "govet",
      "staticcheck",
      "errcheck",
      "gosec",
      "gocritic",
      "nolintlint"
    ],
    "linters_structural": ["gocyclo", "wrapcheck", "dupl"],
    "cosmetic_scope": {
      "enabled_in": ["adapters", "api"],
      "disabled_in": ["domain", "workflow"]
    },
    "rules": [
      "linters_protect_architecture",
      "nolint_requires_reason"
    ]
  },

  "security_practices": {
    "principles": ["least_privilege", "team_isolation"],
    "implementation": {
      "api": "no_provider_credentials",
      "workers": "scoped_credentials",
      "secrets": "domain_resource_with_lifecycle"
    },
    "service_to_service_auth": {
      "pattern": ["signed_tokens", "or_mtls"],
      "rules": [
        "all_internal_http_calls_require_auth_header",
        "tokens_are_short_lived_and_audited",
        "no_cross_team_impersonation_within_single_token",
        "workflows_call_api_as_first_class_clients_not_as_root"
      ]
    },
    "observability_exposure": {
      "metrics": {
        "exposed_on": ["/metrics"],
        "network_boundary": "cluster_internal_only",
        "rules": [
          "no_pii_in_labels_or_metric_names",
          "metrics_endpoints_not_exposed_to_internet",
          "scraping_only_from_trusted_prometheus_instances"
        ]
      },
      "traces": {
        "export": ["otlp"],
        "rules": [
          "trace_backends_are_access_controlled",
          "no_secrets_in_span_attributes",
          "trace_sampling_configured_per_environment"
        ]
      }
    }
  },

  "concurrency_and_locking": {
    "rules": [
      "one_active_workflow_per_application",
      "one_active_workflow_per_application_environment"
    ],
    "implementation": [
      "optimistic_locking_on_state",
      "deterministic_workflow_ids"
    ]
  },

  "domain_versioning_and_migration": {
    "principles": [
      "forward_compatible_domain",
      "no_in_place_state_mutation"
    ],
    "migration_strategy": [
      "migrations_executed_via_workflows",
      "state_transitions_are_auditable"
    ]
  },

  "human_in_the_loop": {
    "pattern": "explicit_pause_and_approval",
    "rules": [
      "approvals_are_domain_events",
      "workflow_can_pause_and_resume",
      "no_manual_side_effects"
    ]
  },

  "bootstrap_and_break_glass": {
    "bootstrap": [
      "explicit_admin_operations",
      "single_use_and_audited"
    ],
    "break_glass": [
      "rare",
      "audited",
      "time_bound",
      "post_mortem_required"
    ]
  },

  "backpressure_and_resilience": {
    "patterns": [
      "rate_limiting_per_provider",
      "circuit_breakers_in_adapters",
      "exponential_backoff"
    ],
    "rules": [
      "no_unbounded_retries",
      "provider_failures_do_not_corrupt_domain"
    ]
  },

  "dashboards_and_alerts": {
    "dashboards": {
      "http_overview": {
        "metrics": [
          "http_requests_total",
          "http_request_duration_seconds"
        ],
        "breakdown": [
          "by_service_and_path",
          "by_status_code"
        ]
      },
      "domain_events_flow": {
        "metrics": ["domain_events_total"],
        "grouping": [
          "by_event_and_result",
          "success_vs_error_rate_per_use_case"
        ]
      }
    },
    "alerts": {
      "http_error_spike": {
        "metric": "http_requests_total",
        "condition": "rate_by_status_5xx_above_threshold",
        "dimension": "per_service_and_path"
      },
      "workflow_failure_spike": {
        "metric": "domain_events_total",
        "filter": "event_matches_workflow_*_failed",
        "condition": "rate_above_baseline"
      },
      "provider_integration_errors": {
        "metric": "domain_events_total",
        "filter": "event_in_[github_repo_created,appenv_side_effect_accepted]",
        "condition": "error_ratio_above_threshold"
      }
    }
  },

  "anti_patterns": [
    "business_logic_in_controllers",
    "direct_provider_calls_from_api",
    "pipelines_as_orchestrators",
    "manual_state_mutation",
    "event_choreography_for_core_flows"
  ],

  "golden_rules": [
    "If_you_see_SDKs_in_domain_you_are_wrong",
    "If_you_retry_in_API_you_are_wrong",
    "If_logs_are_your_source_of_truth_you_are_wrong",
    "If_you_need_manual_steps_the_model_is_missing"
  ]
}