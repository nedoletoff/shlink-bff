from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8", extra="ignore")

    # Server
    http_addr: str = ":8080"

    # Database — defaults to a local SQLite file for zero-config development
    database_url: str = "sqlite+aiosqlite:///./shlink_bff.db"

    # Shlink internal API
    shlink_internal_url: str

    # Feature flags
    feature_user_slug_prefix: bool = False
    feature_user_tag_internal_id: bool = False

    # Optional: CORS allowed origins (comma-separated or JSON list)
    cors_allowed_origins: list[str] = ["*"]

    @property
    def host(self) -> str:
        parts = self.http_addr.rsplit(":", 1)
        return parts[0] if len(parts) == 2 and parts[0] else "0.0.0.0"

    @property
    def port(self) -> int:
        parts = self.http_addr.rsplit(":", 1)
        return int(parts[-1]) if parts[-1].isdigit() else 8080


_settings: Settings | None = None


def get_settings() -> Settings:
    global _settings
    if _settings is None:
        _settings = Settings()
    return _settings
