"""独立 Typer 命令树装配。"""

from __future__ import annotations

import typer

from memgo_cli import __version__
from memgo_cli.cli.registrations.add import register as register_add
from memgo_cli.cli.registrations.agent_rush_add import register as register_agent_rush_add
from memgo_cli.cli.registrations.agent_rush_callback import register as register_agent_rush_callback
from memgo_cli.cli.registrations.agent_rush_search import register as register_agent_rush_search
from memgo_cli.cli.registrations.config_callback import register as register_config_callback
from memgo_cli.cli.registrations.config_get import register as register_config_get
from memgo_cli.cli.registrations.config_set import register as register_config_set
from memgo_cli.cli.registrations.config_show import register as register_config_show
from memgo_cli.cli.registrations.delete import register as register_delete
from memgo_cli.cli.registrations.entity_callback import register as register_entity_callback
from memgo_cli.cli.registrations.entity_delete import register as register_entity_delete
from memgo_cli.cli.registrations.entity_list import register as register_entity_list
from memgo_cli.cli.registrations.event_callback import register as register_event_callback
from memgo_cli.cli.registrations.event_list import register as register_event_list
from memgo_cli.cli.registrations.event_status import register as register_event_status
from memgo_cli.cli.registrations.get import register as register_get
from memgo_cli.cli.registrations.help import register as register_help
from memgo_cli.cli.registrations.identify import register as register_identify
from memgo_cli.cli.registrations.import_cmd import register as register_import_cmd
from memgo_cli.cli.registrations.init import register as register_init
from memgo_cli.cli.registrations.list_cmd import register as register_list_cmd
from memgo_cli.cli.registrations.main_callback import register as register_main_callback
from memgo_cli.cli.registrations.search import register as register_search
from memgo_cli.cli.registrations.status import register as register_status
from memgo_cli.cli.registrations.update import register as register_update
from memgo_cli.cli.registrations.version import register as register_version
from memgo_cli.cli.registrations.whoami_cmd import register as register_whoami_cmd


def create_app() -> typer.Typer:
    """创建独立的命令树，保持原注册顺序。"""
    app = typer.Typer(
        name="memgo",
        help=f"◆ MemGo CLI v{__version__} · Python SDK\n\n   The Memory Layer for AI Agents",
        no_args_is_help=True,
        rich_markup_mode="rich",
        pretty_exceptions_enable=False,
        add_completion=False,
        subcommand_metavar="<command> [options]",
        options_metavar="",
    )
    config_app = typer.Typer(
        name="config",
        help="Manage memgo configuration.",
        no_args_is_help=True,
        rich_markup_mode="rich",
    )
    entity_app = typer.Typer(
        name="entity",
        help="Manage entities.",
        no_args_is_help=True,
        rich_markup_mode="rich",
    )
    event_app = typer.Typer(
        name="event",
        help="Inspect background processing events.",
        no_args_is_help=True,
        rich_markup_mode="rich",
    )
    register_config_callback(config_app)
    register_entity_callback(entity_app)
    register_event_callback(event_app)
    register_main_callback(app)
    register_add(app)
    register_search(app)
    register_get(app)
    register_list_cmd(app)
    register_update(app)
    register_delete(app)
    register_config_show(config_app)
    register_config_get(config_app)
    register_config_set(config_app)
    register_entity_list(entity_app)
    register_entity_delete(entity_app)
    app.add_typer(entity_app, name="entity", rich_help_panel="Management")
    register_event_list(event_app)
    register_event_status(event_app)
    app.add_typer(event_app, name="event", rich_help_panel="Management")
    register_init(app)
    register_identify(app)
    register_whoami_cmd(app)
    agent_rush_app = typer.Typer(
        name="agent-rush",
        help="AGENTRUSH game commands",
        no_args_is_help=True,
        rich_markup_mode="rich",
    )
    register_agent_rush_callback(agent_rush_app)
    register_agent_rush_add(agent_rush_app)
    register_agent_rush_search(agent_rush_app)
    app.add_typer(agent_rush_app, name="agent-rush", rich_help_panel="Setup")
    register_status(app)
    register_version(app)
    register_import_cmd(app)
    register_help(app)
    app.add_typer(config_app, name="config", rich_help_panel="Management")
    return app
