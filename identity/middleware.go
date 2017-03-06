package identity

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo"
)

// MultiTenant extract tenant information from hostname and inject it into current context
func MultiTenant(tenantService TenantService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			hostname := stripPort(c.Request().Host)
			tenant, err := tenantService.GetByDomain(hostname)
			if err == nil {
				c.Set("Tenant", tenant)
				return next(c)
			}

			c.Logger().Infof("Tenant not found for '%s'.", hostname)
			return c.NoContent(http.StatusNotFound)
		}
	}
}

// JwtGetter gets JWT token from cookie and add into context
func JwtGetter() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			if cookie, err := c.Cookie("auth"); err == nil {
				if claims, err := Decode(cookie.Value); err == nil {
					c.Set("Claims", claims)
				} else {
					c.Logger().Error(err)
				}
			}

			return next(c)
		}
	}
}

// JwtSetter sets JWT token into cookie
func JwtSetter() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			query := c.Request().URL.Query()

			jwt := query.Get("jwt")
			if jwt != "" {
				c.SetCookie(&http.Cookie{
					Name:     "auth",
					Value:    jwt,
					HttpOnly: true,
				})

				scheme := "http"
				if c.Request().TLS != nil {
					scheme = "https"
				}

				query.Del("jwt")

				url := scheme + "://" + c.Request().Host + c.Request().URL.Path
				querystring := query.Encode()
				if querystring != "" {
					url += "?" + querystring
				}

				return c.Redirect(http.StatusTemporaryRedirect, url)
			}

			return next(c)
		}
	}
}

// HostChecker checks for a specific host
func HostChecker(baseURL string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			u, _ := url.Parse(baseURL)

			if c.Request().Host != u.Host {
				c.Logger().Errorf("%s is not valid for this operation. Only %s is allowed.", c.Request().Host, u.Host)
				return c.NoContent(http.StatusBadRequest)
			}

			return next(c)
		}
	}
}

func stripPort(hostport string) string {
	colon := strings.IndexByte(hostport, ':')
	if colon == -1 {
		return hostport
	}
	return hostport[:colon]
}
