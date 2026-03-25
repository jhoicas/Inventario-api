package crm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	stdmail "net/mail"
	"sort"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/google/uuid"
	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type IMAPSecretEncryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type customerEmailLookupRepository interface {
	GetByCompanyAndEmail(companyID, email string) (*entity.Customer, error)
}

type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type EmailOAuthConfig struct {
	Google    OAuthProviderConfig
	Microsoft OAuthProviderConfig
}

type EmailUseCase struct {
	accountRepo  repository.EmailAccountRepository
	emailRepo    repository.EmailRepository
	hybridRepo   repository.HybridEmailAccountRepository
	customerRepo customerEmailLookupRepository
	ticketRepo   repository.CRMTicketRepository
	oauthCfg     EmailOAuthConfig
	encryptor    IMAPSecretEncryptor
}

func NewEmailUseCase(
	accountRepo repository.EmailAccountRepository,
	emailRepo repository.EmailRepository,
	hybridRepo repository.HybridEmailAccountRepository,
	customerRepo customerEmailLookupRepository,
	ticketRepo repository.CRMTicketRepository,
	oauthCfg EmailOAuthConfig,
	encryptor IMAPSecretEncryptor,
) *EmailUseCase {
	return &EmailUseCase{
		accountRepo:  accountRepo,
		emailRepo:    emailRepo,
		hybridRepo:   hybridRepo,
		customerRepo: customerRepo,
		ticketRepo:   ticketRepo,
		oauthCfg:     oauthCfg,
		encryptor:    encryptor,
	}
}

func (uc *EmailUseCase) ProcessOAuthAccount(companyID, userID string, in dto.OAuthEmailAccountRequest) (*dto.EmailAccountConfigResponse, error) {
	if strings.TrimSpace(companyID) == "" || strings.TrimSpace(userID) == "" {
		return nil, domain.ErrUnauthorized
	}
	if uc.hybridRepo == nil {
		return nil, fmt.Errorf("hybrid email account repository no configurado")
	}

	provider := strings.ToLower(strings.TrimSpace(in.Provider))
	if provider == "" || strings.TrimSpace(in.AuthCode) == "" {
		return nil, domain.ErrInvalidInput
	}

	oauthConfig, scopes, err := uc.oauthConfigFor(provider, strings.TrimSpace(in.RedirectURI))
	if err != nil {
		return nil, err
	}

	tok, err := oauthConfig.Exchange(context.Background(), strings.TrimSpace(in.AuthCode))
	if err != nil {
		return nil, fmt.Errorf("intercambiar auth code de %s: %w", provider, err)
	}

	_ = scopes // alcance usado para construir config de intercambio

	emailAddress := strings.TrimSpace(strings.ToLower(in.EmailAddress))
	if emailAddress == "" {
		if tokenEmail, ok := tok.Extra("email").(string); ok {
			emailAddress = strings.TrimSpace(strings.ToLower(tokenEmail))
		}
	}
	if emailAddress == "" {
		return nil, domain.ErrInvalidInput
	}

	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	item := &entity.EmailAccountConfig{
		ID:           uuid.New().String(),
		UserID:       userID,
		CompanyID:    companyID,
		Provider:     provider,
		EmailAddress: emailAddress,
		AccessToken:  strings.TrimSpace(tok.AccessToken),
		RefreshToken: strings.TrimSpace(tok.RefreshToken),
		IsActive:     isActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := uc.hybridRepo.Save(context.Background(), item); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "uq_email_accounts_company_email") {
			return nil, domain.ErrDuplicate
		}
		return nil, err
	}
	resp := toEmailAccountConfigResponse(item)
	return &resp, nil
}

func (uc *EmailUseCase) SaveCustomAccount(companyID, userID string, in dto.CustomEmailAccountRequest) (*dto.EmailAccountConfigResponse, error) {
	if strings.TrimSpace(companyID) == "" || strings.TrimSpace(userID) == "" {
		return nil, domain.ErrUnauthorized
	}
	if uc.hybridRepo == nil {
		return nil, fmt.Errorf("hybrid email account repository no configurado")
	}

	imapHost := normalizeIMAPServer(in.ImapHost)
	if strings.TrimSpace(in.EmailAddress) == "" || imapHost == "" || in.ImapPort <= 0 || strings.TrimSpace(in.AppPassword) == "" {
		return nil, domain.ErrInvalidInput
	}

	encryptedPassword, err := uc.encryptor.Encrypt(strings.TrimSpace(in.AppPassword))
	if err != nil {
		return nil, err
	}

	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	item := &entity.EmailAccountConfig{
		ID:           uuid.New().String(),
		UserID:       userID,
		CompanyID:    companyID,
		Provider:     "custom",
		EmailAddress: strings.TrimSpace(strings.ToLower(in.EmailAddress)),
		ImapHost:     imapHost,
		ImapPort:     in.ImapPort,
		SmtpHost:     strings.TrimSpace(in.SmtpHost),
		SmtpPort:     in.SmtpPort,
		AppPassword:  encryptedPassword,
		IsActive:     isActive,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := uc.hybridRepo.Save(context.Background(), item); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "uq_email_accounts_company_email") {
			return nil, domain.ErrDuplicate
		}
		return nil, err
	}
	resp := toEmailAccountConfigResponse(item)
	return &resp, nil
}

func (uc *EmailUseCase) CreateAccount(companyID string, in dto.CreateEmailAccountRequest) (*dto.EmailAccountResponse, error) {
	if strings.TrimSpace(companyID) == "" {
		return nil, domain.ErrUnauthorized
	}
	imapServer := normalizeIMAPServer(in.IMAPServer)
	if strings.TrimSpace(in.EmailAddress) == "" || imapServer == "" || in.IMAPPort <= 0 || strings.TrimSpace(in.Password) == "" {
		return nil, domain.ErrInvalidInput
	}

	encrypted, err := uc.encryptor.Encrypt(in.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}
	account := &entity.EmailAccount{
		ID:           uuid.New().String(),
		CompanyID:    companyID,
		EmailAddress: strings.TrimSpace(strings.ToLower(in.EmailAddress)),
		IMAPServer:   imapServer,
		IMAPPort:     in.IMAPPort,
		Password:     encrypted,
		IsActive:     isActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := uc.accountRepo.Create(account); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "uq_email_accounts_company_email") {
			return nil, domain.ErrDuplicate
		}
		return nil, err
	}
	return toEmailAccountResponse(account), nil
}

func (uc *EmailUseCase) UpdateAccount(companyID, id string, in dto.UpdateEmailAccountRequest) (*dto.EmailAccountResponse, error) {
	if strings.TrimSpace(companyID) == "" || strings.TrimSpace(id) == "" {
		return nil, domain.ErrInvalidInput
	}
	acc, err := uc.accountRepo.GetByID(companyID, id)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, domain.ErrNotFound
	}

	if in.EmailAddress != nil {
		acc.EmailAddress = strings.TrimSpace(strings.ToLower(*in.EmailAddress))
	}
	if in.IMAPServer != nil {
		acc.IMAPServer = normalizeIMAPServer(*in.IMAPServer)
	}
	if in.IMAPPort != nil {
		acc.IMAPPort = *in.IMAPPort
	}
	if in.Password != nil {
		encrypted, err := uc.encryptor.Encrypt(*in.Password)
		if err != nil {
			return nil, err
		}
		acc.Password = encrypted
	}
	if in.IsActive != nil {
		acc.IsActive = *in.IsActive
	}
	if strings.TrimSpace(acc.EmailAddress) == "" || strings.TrimSpace(acc.IMAPServer) == "" || acc.IMAPPort <= 0 {
		return nil, domain.ErrInvalidInput
	}

	acc.UpdatedAt = time.Now().UTC()
	if err := uc.accountRepo.Update(acc); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "uq_email_accounts_company_email") {
			return nil, domain.ErrDuplicate
		}
		return nil, err
	}
	return toEmailAccountResponse(acc), nil
}

func (uc *EmailUseCase) DeleteAccount(companyID, id string) error {
	if strings.TrimSpace(companyID) == "" || strings.TrimSpace(id) == "" {
		return domain.ErrInvalidInput
	}
	acc, err := uc.accountRepo.GetByID(companyID, id)
	if err != nil {
		return err
	}
	if acc == nil {
		return domain.ErrNotFound
	}
	return uc.accountRepo.Delete(companyID, id)
}

func (uc *EmailUseCase) GetAccount(companyID, id string) (*dto.EmailAccountResponse, error) {
	if strings.TrimSpace(companyID) == "" || strings.TrimSpace(id) == "" {
		return nil, domain.ErrInvalidInput
	}
	acc, err := uc.accountRepo.GetByID(companyID, id)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, domain.ErrNotFound
	}
	return toEmailAccountResponse(acc), nil
}

func (uc *EmailUseCase) ListAccounts(companyID string, limit, offset int) ([]dto.EmailAccountResponse, error) {
	if strings.TrimSpace(companyID) == "" {
		return nil, domain.ErrUnauthorized
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	list, err := uc.accountRepo.ListByCompany(companyID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]dto.EmailAccountResponse, 0, len(list))
	for _, item := range list {
		out = append(out, *toEmailAccountResponse(item))
	}
	return out, nil
}

func (uc *EmailUseCase) TestConnection(companyID, id string) (*dto.TestIMAPConnectionResponse, error) {
	if strings.TrimSpace(companyID) == "" || strings.TrimSpace(id) == "" {
		return nil, domain.ErrInvalidInput
	}
	acc, err := uc.accountRepo.GetByID(companyID, id)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, domain.ErrNotFound
	}
	if err := uc.testIMAPConnection(acc); err != nil {
		return &dto.TestIMAPConnectionResponse{Success: false, Message: err.Error()}, nil
	}
	return &dto.TestIMAPConnectionResponse{Success: true, Message: "conexión IMAP exitosa"}, nil
}

func (uc *EmailUseCase) TestConnectionBeforeSave(companyID string, in dto.CreateEmailAccountRequest) (*dto.TestIMAPConnectionResponse, error) {
	if strings.TrimSpace(companyID) == "" {
		return nil, domain.ErrUnauthorized
	}
	imapServer := normalizeIMAPServer(in.IMAPServer)
	if strings.TrimSpace(in.EmailAddress) == "" || imapServer == "" || in.IMAPPort <= 0 || strings.TrimSpace(in.Password) == "" {
		return nil, domain.ErrInvalidInput
	}

	if err := testIMAPCredentials(strings.TrimSpace(strings.ToLower(in.EmailAddress)), strings.TrimSpace(in.Password), imapServer, in.IMAPPort); err != nil {
		return &dto.TestIMAPConnectionResponse{Success: false, Message: err.Error()}, nil
	}
	return &dto.TestIMAPConnectionResponse{Success: true, Message: "conexión IMAP exitosa"}, nil
}

func (uc *EmailUseCase) ListEmails(companyID, customerID string, isRead *bool, limit, offset int) (*dto.EmailListResponse, error) {
	if strings.TrimSpace(companyID) == "" {
		return nil, domain.ErrUnauthorized
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	items, total, err := uc.emailRepo.ListByCompany(repository.EmailListFilter{
		CompanyID:  companyID,
		CustomerID: strings.TrimSpace(customerID),
		IsRead:     isRead,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return nil, err
	}

	out := make([]dto.EmailResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toEmailResponse(item))
	}
	return &dto.EmailListResponse{Items: out, Total: total, Limit: limit, Offset: offset}, nil
}

func (uc *EmailUseCase) GetEmailAndMarkAsRead(companyID, id string) (*dto.EmailResponse, error) {
	if strings.TrimSpace(companyID) == "" || strings.TrimSpace(id) == "" {
		return nil, domain.ErrInvalidInput
	}
	em, err := uc.emailRepo.GetByID(companyID, id)
	if err != nil {
		return nil, err
	}
	if em == nil {
		return nil, domain.ErrNotFound
	}
	if !em.IsRead {
		if err := uc.emailRepo.MarkAsRead(companyID, id); err != nil {
			return nil, err
		}
		em.IsRead = true
	}
	resp := toEmailResponse(em)
	return &resp, nil
}

func (uc *EmailUseCase) CreateTicketFromEmail(companyID, userID, emailID string) (*dto.CreateTicketFromEmailResponse, error) {
	if strings.TrimSpace(companyID) == "" || strings.TrimSpace(emailID) == "" {
		return nil, domain.ErrInvalidInput
	}
	em, err := uc.emailRepo.GetByID(companyID, emailID)
	if err != nil {
		return nil, err
	}
	if em == nil {
		return nil, domain.ErrNotFound
	}
	if strings.TrimSpace(em.CustomerID) == "" {
		return nil, domain.ErrConflict
	}

	description := strings.TrimSpace(em.BodyText)
	if description == "" {
		description = strings.TrimSpace(em.BodyHTML)
	}
	if description == "" {
		description = "Ticket generado desde correo sin contenido de cuerpo"
	}

	now := time.Now().UTC()
	ticket := &entity.CRMTicket{
		ID:          uuid.New().String(),
		CompanyID:   companyID,
		CustomerID:  em.CustomerID,
		Subject:     em.Subject,
		Description: description,
		Status:      entity.TicketStatusOpen,
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := uc.ticketRepo.Create(ticket); err != nil {
		return nil, err
	}
	return &dto.CreateTicketFromEmailResponse{TicketID: ticket.ID, Status: ticket.Status}, nil
}

func (uc *EmailUseCase) SyncActiveAccounts(ctx context.Context) {
	accounts, err := uc.accountRepo.ListActive()
	if err != nil {
		return
	}
	for _, account := range accounts {
		_ = uc.syncAccount(ctx, account)
	}
}

func (uc *EmailUseCase) testIMAPConnection(acc *entity.EmailAccount) error {
	password, err := uc.encryptor.Decrypt(acc.Password)
	if err != nil {
		return fmt.Errorf("descifrar credenciales: %w", err)
	}
	return testIMAPCredentials(acc.EmailAddress, password, acc.IMAPServer, acc.IMAPPort)
}

func testIMAPCredentials(emailAddress, password, imapServer string, imapPort int) error {
	addr := net.JoinHostPort(normalizeIMAPServer(imapServer), fmt.Sprintf("%d", imapPort))
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return fmt.Errorf("dial IMAP: %w", err)
	}
	defer c.Logout()
	if err := c.Login(strings.TrimSpace(strings.ToLower(emailAddress)), password); err != nil {
		return fmt.Errorf("login IMAP: %w", err)
	}
	if _, err := c.Select("INBOX", false); err != nil {
		return fmt.Errorf("select INBOX: %w", err)
	}
	return nil
}

func (uc *EmailUseCase) syncAccount(ctx context.Context, acc *entity.EmailAccount) error {
	password, err := uc.encryptor.Decrypt(acc.Password)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(normalizeIMAPServer(acc.IMAPServer), fmt.Sprintf("%d", acc.IMAPPort))
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		return err
	}
	defer c.Logout()
	if err := c.Login(acc.EmailAddress, password); err != nil {
		return err
	}
	_, err = c.Select("INBOX", false)
	if err != nil {
		return err
	}

	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	ids, err := c.Search(criteria)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) > 100 {
		ids = ids[len(ids)-100:]
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchInternalDate, section.FetchItem()}
	messages := make(chan *imap.Message, 20)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, items, messages)
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-messages:
			if !ok {
				if err := <-done; err != nil {
					return err
				}
				return nil
			}
			if msg == nil {
				continue
			}
			if err := uc.persistFetchedMessage(acc, msg, section); err != nil {
				continue
			}
		}
	}
}

func normalizeIMAPServer(value string) string {
	server := strings.TrimSpace(value)
	server = strings.TrimPrefix(server, "imaps://")
	server = strings.TrimPrefix(server, "imap://")
	server = strings.TrimPrefix(server, "https://")
	server = strings.TrimPrefix(server, "http://")
	server = strings.TrimRight(server, "/")
	return strings.TrimSpace(server)
}

func toEmailAccountConfigResponse(item *entity.EmailAccountConfig) dto.EmailAccountConfigResponse {
	if item == nil {
		return dto.EmailAccountConfigResponse{}
	}
	return dto.EmailAccountConfigResponse{
		ID:           item.ID,
		UserID:       item.UserID,
		CompanyID:    item.CompanyID,
		Provider:     item.Provider,
		EmailAddress: item.EmailAddress,
		ImapHost:     item.ImapHost,
		ImapPort:     item.ImapPort,
		SmtpHost:     item.SmtpHost,
		SmtpPort:     item.SmtpPort,
		IsActive:     item.IsActive,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

func (uc *EmailUseCase) oauthConfigFor(provider, redirectURI string) (*oauth2.Config, []string, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" {
		return nil, nil, domain.ErrInvalidInput
	}

	scopes := []string{"openid", "profile", "email"}
	var endpoint oauth2.Endpoint
	var cfg OAuthProviderConfig

	switch provider {
	case "google":
		cfg = uc.oauthCfg.Google
		endpoint = google.Endpoint
		scopes = []string{"openid", "profile", "email", "https://mail.google.com/"}
	case "microsoft":
		cfg = uc.oauthCfg.Microsoft
		endpoint = oauth2.Endpoint{
			AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		}
		scopes = []string{"openid", "profile", "email", "offline_access", "https://outlook.office.com/IMAP.AccessAsUser.All", "https://outlook.office.com/SMTP.Send"}
	default:
		return nil, nil, domain.ErrInvalidInput
	}

	clientID := strings.TrimSpace(cfg.ClientID)
	clientSecret := strings.TrimSpace(cfg.ClientSecret)
	finalRedirectURI := strings.TrimSpace(redirectURI)
	if finalRedirectURI == "" {
		finalRedirectURI = strings.TrimSpace(cfg.RedirectURL)
	}

	if clientID == "" || clientSecret == "" || finalRedirectURI == "" {
		return nil, nil, fmt.Errorf("configuración OAuth incompleta para provider %s", provider)
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  finalRedirectURI,
		Endpoint:     endpoint,
		Scopes:       scopes,
	}, scopes, nil
}

func (uc *EmailUseCase) persistFetchedMessage(acc *entity.EmailAccount, msg *imap.Message, section *imap.BodySectionName) error {
	rawBody := msg.GetBody(section)
	if rawBody == nil {
		return nil
	}
	payload, err := io.ReadAll(rawBody)
	if err != nil {
		return err
	}

	parsed, err := stdmail.ReadMessage(bytes.NewReader(payload))
	if err != nil {
		return err
	}

	messageID := strings.TrimSpace(parsed.Header.Get("Message-ID"))
	messageID = strings.Trim(messageID, "<>")
	if messageID == "" && msg.Envelope != nil {
		messageID = strings.TrimSpace(msg.Envelope.MessageId)
	}
	if messageID == "" {
		messageID = fmt.Sprintf("fallback-%s-%d", acc.ID, msg.SeqNum)
	}

	existing, err := uc.emailRepo.GetByAccountAndMessageID(acc.ID, messageID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	fromAddress, toAddress := extractAddresses(msg, parsed.Header)
	subject := strings.TrimSpace(parsed.Header.Get("Subject"))
	if subject == "" && msg.Envelope != nil {
		subject = msg.Envelope.Subject
	}
	if subject == "" {
		subject = "(sin asunto)"
	}

	bodyText, bodyHTML, attachments := extractBodiesAndAttachments(parsed)
	receivedAt := msg.InternalDate
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}

	var customerID string
	if fromAddress != "" {
		customer, err := uc.customerRepo.GetByCompanyAndEmail(acc.CompanyID, fromAddress)
		if err == nil && customer != nil {
			customerID = customer.ID
		}
	}

	email := &entity.Email{
		ID:          uuid.New().String(),
		AccountID:   acc.ID,
		MessageID:   messageID,
		CustomerID:  customerID,
		FromAddress: fromAddress,
		ToAddress:   toAddress,
		Subject:     subject,
		BodyHTML:    bodyHTML,
		BodyText:    bodyText,
		ReceivedAt:  receivedAt.UTC(),
		IsRead:      false,
		CreatedAt:   time.Now().UTC(),
	}
	return uc.emailRepo.Create(email, attachments)
}

func extractAddresses(msg *imap.Message, header stdmail.Header) (string, string) {
	fromAddress := ""
	toAddress := ""

	if fromList, err := stdmail.ParseAddressList(header.Get("From")); err == nil && len(fromList) > 0 {
		fromAddress = strings.ToLower(strings.TrimSpace(fromList[0].Address))
	} else if msg != nil && msg.Envelope != nil && len(msg.Envelope.From) > 0 {
		from := msg.Envelope.From[0]
		fromAddress = strings.ToLower(strings.TrimSpace(from.MailboxName + "@" + from.HostName))
	}

	if toList, err := stdmail.ParseAddressList(header.Get("To")); err == nil && len(toList) > 0 {
		vals := make([]string, 0, len(toList))
		for _, item := range toList {
			vals = append(vals, strings.ToLower(strings.TrimSpace(item.Address)))
		}
		toAddress = strings.Join(vals, ",")
	} else if msg != nil && msg.Envelope != nil && len(msg.Envelope.To) > 0 {
		vals := make([]string, 0, len(msg.Envelope.To))
		for _, item := range msg.Envelope.To {
			vals = append(vals, strings.ToLower(strings.TrimSpace(item.MailboxName+"@"+item.HostName)))
		}
		toAddress = strings.Join(vals, ",")
	}

	return fromAddress, toAddress
}

func extractBodiesAndAttachments(message *stdmail.Message) (string, string, []entity.EmailAttachment) {
	contentType := message.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = "text/plain"
	}

	if strings.HasPrefix(strings.ToLower(mediaType), "multipart/") {
		reader := multipart.NewReader(message.Body, params["boundary"])
		var textParts []string
		var htmlParts []string
		attachments := make([]entity.EmailAttachment, 0)
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			partBytes, err := io.ReadAll(part)
			if err != nil {
				continue
			}
			partCT := part.Header.Get("Content-Type")
			partMediaType, _, _ := mime.ParseMediaType(partCT)
			dispType, dispParams, _ := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
			fileName := part.FileName()
			if fileName == "" {
				fileName = dispParams["filename"]
			}
			if strings.EqualFold(dispType, "attachment") || fileName != "" {
				if partMediaType == "" {
					partMediaType = "application/octet-stream"
				}
				attachments = append(attachments, entity.EmailAttachment{
					ID:       uuid.New().String(),
					FileName: fileName,
					FileURL:  "",
					MIMEType: partMediaType,
					Size:     len(partBytes),
				})
				continue
			}
			if strings.HasPrefix(strings.ToLower(partMediaType), "text/html") {
				htmlParts = append(htmlParts, string(partBytes))
				continue
			}
			textParts = append(textParts, string(partBytes))
		}
		return strings.Join(textParts, "\n"), strings.Join(htmlParts, "\n"), attachments
	}

	bodyBytes, _ := io.ReadAll(message.Body)
	if strings.Contains(strings.ToLower(mediaType), "text/html") {
		return "", string(bodyBytes), nil
	}
	return string(bodyBytes), "", nil
}

func toEmailAccountResponse(item *entity.EmailAccount) *dto.EmailAccountResponse {
	return &dto.EmailAccountResponse{
		ID:           item.ID,
		CompanyID:    item.CompanyID,
		EmailAddress: item.EmailAddress,
		IMAPServer:   item.IMAPServer,
		IMAPPort:     item.IMAPPort,
		IsActive:     item.IsActive,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

func toEmailResponse(item *entity.Email) dto.EmailResponse {
	attachments := make([]dto.EmailAttachmentResponse, 0, len(item.Attachments))
	for _, a := range item.Attachments {
		attachments = append(attachments, dto.EmailAttachmentResponse{
			ID:       a.ID,
			FileName: a.FileName,
			FileURL:  a.FileURL,
			MIMEType: a.MIMEType,
			Size:     a.Size,
		})
	}
	return dto.EmailResponse{
		ID:          item.ID,
		AccountID:   item.AccountID,
		CustomerID:  item.CustomerID,
		MessageID:   item.MessageID,
		FromAddress: item.FromAddress,
		ToAddress:   item.ToAddress,
		Subject:     item.Subject,
		BodyHTML:    item.BodyHTML,
		BodyText:    item.BodyText,
		ReceivedAt:  item.ReceivedAt,
		IsRead:      item.IsRead,
		CreatedAt:   item.CreatedAt,
		Attachments: attachments,
	}
}
