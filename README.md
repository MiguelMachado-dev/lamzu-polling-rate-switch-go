# LAMZU Polling Rate Auto-Switch (Go)

Versão em Go do automator para mouse LAMZU que ajusta automaticamente o polling rate baseado nos jogos em execução.

## Características

- ✅ Executável único (.exe) sem dependências
- ✅ Baixo uso de recursos (Go nativo)
- ✅ Sem GUI - CLI/Service apenas
- ✅ Configuração via arquivo YAML
- ✅ Suporte a serviço Windows
- ✅ Protocolo HID nativo do LAMZU Maya X 8K
- 🆕 **Windows HID API Nativa**: Implementação direta usando hid.dll e setupapi.dll
- 🆕 **Melhor detecção de dispositivos**: Usa SetupDi APIs para máxima confiabilidade

## Instalação

1. Baixe o executável `lamzu-automator.exe`
2. Execute `build.bat` para compilar do código fonte (opcional)
3. Configure os jogos em `config.yaml`
4. Execute como administrador

## Uso

### Modo Interativo
```bash
# Executar normalmente
lamzu-automator.exe

# Executar com arquivo de configuração personalizado
lamzu-automator.exe -c minha-config.yaml

# Executar com output verbose
lamzu-automator.exe -v
```

### Modo Daemon/Service
```bash
# Executar como daemon (background)
lamzu-automator.exe -d

# Instalar como serviço Windows
install.bat

# Remover serviço Windows
uninstall.bat
```

### Comandos Manuais
```bash
# Definir polling rate manualmente
lamzu-automator.exe set 2000

# Listar polling rates disponíveis
lamzu-automator.exe list

# Ajuda
lamzu-automator.exe --help
```

## Configuração

Edite o arquivo `config.yaml`:

```yaml
default_polling_rate: 1000  # Polling rate padrão (desktop)
game_polling_rate: 2000     # Polling rate para jogos
check_interval: 2s          # Intervalo de verificação
games:                      # Lista de jogos (processos)
  - HuntGame.exe
  - DuneSandbox-Wi.exe
  - eldenring.exe
  - cs2.exe
  - valorant.exe
```

## Requisitos

- Windows 10/11
- Mouse LAMZU Maya X 8K
- Executar como Administrador (necessário para HID)

## Vantagens vs Versão Node.js

- **Tamanho**: ~5MB vs ~80MB (16x menor)
- **Memória**: ~10MB vs ~50MB (5x menos)
- **Startup**: Instantâneo vs ~2 segundos
- **Dependências**: Zero vs Node.js + Electron
- **Segurança**: Executável nativo vs JavaScript

## Compilação

```bash
# Instalar Go 1.21+
go mod tidy
go build -ldflags="-s -w" -o lamzu-automator.exe .
```

## Implementação HID Nativa

O aplicativo utiliza APIs Windows nativas diretamente para máxima confiabilidade:

**Windows API Nativa**: Usa `hid.dll`, `setupapi.dll` diretamente
- Descoberta de dispositivos mais confiável
- Usa `HidD_GetHidGuid`, `SetupDiGetClassDevs`
- Filtragem por interface (interface 2 para LAMZU)
- Comandos via `HidD_SetFeature` para feature reports
- Melhor integração com o sistema Windows

Execute com `-v` para ver detalhes da descoberta de dispositivos:
```bash
lamzu-automator.exe debug -v
```

## Troubleshooting

**Erro "device not found"**:
- Execute como Administrador
- Verifique se o mouse está conectado
- Confirme que é um LAMZU Maya X 8K
- Teste: `lamzu-automator.exe debug -v`

**Polling rate não muda**:
- Reinicie o mouse (desconecte/reconecte)
- Verifique se nenhum outro software está controlando o mouse
- Confirme que o processo do jogo está na lista de configuração

**Debug avançado**:
```bash
# Listar todos os dispositivos HID
lamzu-automator.exe list-all -v

# Debug completo com APIs nativas
lamzu-automator.exe debug -v
```